import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
  ProFormText,
  ProFormSwitch,
  ProFormSelect,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_ApiAuditLog as ApiAuditLog } from '@/api/generated/admin/service/v1';
import { useCreateApiAuditLog, useUpdateApiAuditLog } from '@/api/hooks/api-audit-log';
import { getRequestMethodOptions } from '../constants';

interface ApiAuditLogDrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: ApiAuditLog;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * ApiAuditLog 编辑/创建抽屉组件
 */
const ApiAuditLogDrawer: React.FC<ApiAuditLogDrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('api-audit-log');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);

  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {

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
        isSuccess: false,
            }
      }
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading,
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
        name="duration"
        label={t('duration')}
        placeholder={t('durationPlaceholder')}
        rules={[{ required: true, message: t('requiredDuration') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormText
        name="ipAddress"
        label={t('ipAddress')}
        placeholder={t('ipAddressPlaceholder')}
        rules={[{ required: true, message: t('requiredIpAddress') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormSwitch
        name="isSuccess"
        label={t('isSuccess')}
        placeholder={t('isSuccessPlaceholder')}
      />

      <ProFormText
        name="operatorName"
        label={t('operatorName')}
        placeholder={t('operatorNamePlaceholder')}
        rules={[{ required: true, message: t('requiredOperatorName') }]}
        fieldProps={{ allowClear: true }}
      />

      <ProFormSelect
        name="requestMethod"
        label={t('requestMethod')}
        placeholder={t('requestMethodPlaceholder')}
        rules={[{ required: true, message: t('requiredRequestMethod') }]}
        options={getRequestMethodOptions(t)}
      />

      <ProFormText
        name="requestPath"
        label={t('requestPath')}
        placeholder={t('requestPathPlaceholder')}
        rules={[{ required: true, message: t('requiredRequestPath') }]}
        fieldProps={{ allowClear: true }}
      />
    </DrawerForm>
  );
};

export default ApiAuditLogDrawer;
