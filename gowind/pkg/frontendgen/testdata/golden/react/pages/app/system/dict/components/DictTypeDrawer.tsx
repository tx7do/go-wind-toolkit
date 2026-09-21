import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSwitch,
  ProFormTextArea,
  ProFormDigit,
  ProFormRadio,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { dictservicev1_DictType as DictType } from '@/api/generated/admin/service/v1';
import { useCreateDictType, useUpdateDictType } from '@/api/hooks/dict-type';
import { getStatusOptions } from '../constants';

interface DictTypeDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: DictType;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * DictType 编辑/创建抽屉组件
 */
const DictTypeDrawer: React.FC<DictTypeDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('dict-type');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  const createMutation = useCreateDictType({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listDictTypes'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });

  const updateMutation = useUpdateDictType({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['listDictTypes'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });

  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
      if (mode === 'create') {
        await createMutation.mutateAsync({ data: values });
      } else if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values });
      }
    } finally {
      setConfirmLoading(false);
    }
  };

  return (
    <DrawerForm
      formRef={formRef}
      title={mode === 'create' ? t('create') : t('edit')}
      open={open}
      onOpenChange={(visible) => {
        if (!visible) {
          formRef.current?.resetFields();
          onClose();
        }
      }}
      initialValues={
        mode === 'edit'
          ? { ...data }
          : {
        isEnabled: true,
        sortOrder: 1,
        status: 'ON',
            }
      }
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading || createMutation.isPending || updateMutation.isPending,
        },
        resetButtonProps: {
          onClick: onClose,
        },
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
        width: 480,
      }}
    >
      <ProFormText
        name="code"
        label={t('code')}
        placeholder={t('codePlaceholder')}
        rules={[{ required: true, message: t('requiredCode') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormSwitch
        name="isEnabled"
        label={t('isEnabled')}
        placeholder={t('isEnabledPlaceholder')}
      />

      <ProFormText
        name="name"
        label={t('name')}
        placeholder={t('namePlaceholder')}
        rules={[{ required: true, message: t('requiredName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormTextArea
        name="remark"
        label={t('remark')}
        placeholder={t('remarkPlaceholder')}
        fieldProps={{ allowClear: true, rows: 2 }}
      />

      <ProFormDigit
        name="sortOrder"
        label={t('sortOrder')}
        placeholder={t('sortOrderPlaceholder')}
        fieldProps={{ precision: 0, min: 0 }}
      />

      <ProFormRadio.Group
        name="status"
        label={t('status')}
        placeholder={t('statusPlaceholder')}
        rules={[{ required: true, message: t('requiredStatus') }]}
        options={getStatusOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}
      />
    </DrawerForm>
  );
};

export default DictTypeDrawer;
