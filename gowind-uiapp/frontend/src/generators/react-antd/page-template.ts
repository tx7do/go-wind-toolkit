/**
 * React Ant Design Pro - 页面代码生成器
 * 生成 ProTable 列表页 index.tsx 和 DrawerForm 编辑抽屉组件
 */
import type { ParsedService, ParsedField } from '../../utils/openapi-parser'
import { toPascalCase, getCrudPaths } from '../../utils/openapi-parser'

export interface ReactPageTemplateOptions {
  service: ParsedService
  /** 模块路径名（如 permission/role） */
  modulePath?: string
  /** 服务名（如 admin） */
  serviceName?: string
}

// ==============================
// 字段类型推断辅助函数
// ==============================

/**
 * 获取搜索字段（排除 id、数组、长文本等字段）
 */
function getSearchFields(fields: ParsedField[]): ParsedField[] {
  const skipFields = new Set(['id', 'description', 'remark', 'sortOrder', 'isDefault', 'isEnabled'])
  return fields.filter(f => !skipFields.has(f.name) && !f.isArray && !f.isDate)
}

/**
 * 根据 OpenAPI 字段生成 ProTable 列定义
 */
function generateColumns(service: ParsedService, modelPascal: string, _fileName: string): string {
  const tableFields = service.fields.filter(f => !['id'].includes(f.name) && !f.isArray)
  const lines: string[] = []

  // 序号列
  lines.push(`    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || TABLE.DEFAULT_PAGE_SIZE;
        return (page - 1) * pageSize + index + 1;
      },
    },`)

  for (const field of tableFields) {
    const lower = field.name.toLowerCase()

    // 布尔字段
    if (field.isBoolean) {
      lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 100,
      hideInSearch: true,
      render: (_, record) => {
        const val = record.${field.name} as boolean;
        return <Tag color={val ? 'success' : 'error'}>{val ? t('yes') : t('no')}</Tag>;
      },
    },`)
      continue
    }

    // 枚举字段
    if (field.isEnum && field.enumValues) {
      const hasStatus = lower.includes('status')
      if (hasStatus) {
        lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: getStatusOptions(t),
      },
      render: (_, record) => {
        const statusMap = getStatusMap(t);
        const status = record.${field.name} as keyof typeof statusMap;
        const config = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },`)
      } else {
        lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 120,
      hideInSearch: true,
    },`)
      }
      continue
    }

    // 日期字段
    if (field.isDate) {
      lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
    },`)
      continue
    }

    // 排序字段
    if (field.isInteger && lower.includes('sort')) {
      lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 100,
      hideInSearch: true,
    },`)
      continue
    }

    // 描述/备注字段
    if (lower.includes('description') || lower.includes('remark')) {
      lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      hideInSearch: true,
      ellipsis: true,
    },`)
      continue
    }

    // 搜索字段
    const isSearchField = getSearchFields(service.fields).some(sf => sf.name === field.name)

    if (isSearchField) {
      lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 150,
    },`)
    } else {
      lines.push(`    {
      title: t('${field.name}'),
      dataIndex: '${field.name}',
      width: 150,
      hideInSearch: true,
    },`)
    }
  }

  // 操作列
  lines.push(`    {
      title: t('action'),
      valueType: 'option',
      width: 100,
      fixed: 'right',
      render: (_, record) => [
        <a
          key="edit"
          onClick={() => {
            setDrawerMode('edit');
            setSelected${modelPascal}(record);
            setDrawerOpen(true);
          }}
        >
          <EditOutlined />
        </a>,
        <Popconfirm
          key="delete"
          title={t('deleteConfirmTitle')}
          description={t('deleteConfirmDesc', { moduleName: t('moduleName') })}
          onConfirm={() => record.id && deleteMutation.mutate({ id: record.id })}
          okText={t('common:button.ok')}
          cancelText={t('common:button.cancel')}
        >
          <a style={{ color: '#ff4d4f' }}><DeleteOutlined /></a>
        </Popconfirm>,
      ],
    },`)

  return lines.join('\n')
}

// ==============================
// 列表页面 index.tsx
// ==============================

export function generatePageCode(options: ReactPageTemplateOptions): string {
  const {
    service,
    modulePath = service.kebabName,
    serviceName = 'admin',
  } = options

  const modelPascal = toPascalCase(service.modelName)
  const fileName = service.kebabName.replace(/-/g, '')
  const crudPaths = getCrudPaths(service)
  const hasList = !!crudPaths.list

  if (!hasList) return `// ${service.tagName} 没有 List 操作，无法生成列表页面`

  const hasStatusEnum = service.fields.some(f => f.isEnum && f.name.toLowerCase().includes('status'))

  const columnsCode = generateColumns(service, modelPascal, fileName)

  let code = `import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Popconfirm, Tag, App } from 'antd';
import { EditOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { ${fileName}_${modelPascal} as ${modelPascal} } from '@/api/generated/${serviceName}/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchList${modelPascal}s, useDelete${modelPascal} } from '@/api/hooks/${fileName}';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
${hasStatusEnum ? `import { getStatusMap, getStatusOptions } from './constants';\n` : ''}import ${modelPascal}Drawer from './components/${modelPascal}Drawer';

/**
 * ${service.description}
 */
const ${modelPascal}Management = () => {
  const { t } = useTranslation('${fileName}');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'edit'>('create');
  const [selected${modelPascal}, setSelected${modelPascal}] = useState<${modelPascal} | undefined>();

  // 删除操作
  const deleteMutation = useDelete${modelPascal}({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      actionRef.current?.reload();
      queryClient.invalidateQueries({ queryKey: ['list${modelPascal}s'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 列配置
  const columns: ProColumns<${modelPascal}>[] = [
${columnsCode}
  ];

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ProTable<${modelPascal}>
            actionRef={actionRef}
            columns={columns}
            request={async (params, _sorter, _filter) => {
              try {
                const query = new PaginationQuery({
                  paging: {
                    page: params.current || 1,
                    pageSize: params.pageSize || TABLE.DEFAULT_PAGE_SIZE,
                  },
                  formValues: Object.fromEntries(
                    Object.entries(params).filter(
                      ([key]) => !['current', 'pageSize'].includes(key),
                    ),
                  ),
                });

                const response = await fetchList${modelPascal}s(query);

                return {
                  data: response.items || [],
                  total: response.total || 0,
                  success: true,
                };
              } catch (error: any) {
                message.error(error.message || t('fetchFailed'));
                return { data: [], total: 0, success: false };
              }
            }}
            rowKey="id"
            search={{
              labelWidth: 'auto',
              defaultCollapsed: false,
            }}
            pagination={{
              defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
              showSizeChanger: true,
              showQuickJumper: true,
            }}
            toolBarRender={() => [
              <Button
                key="create"
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setDrawerMode('create');
                  setSelected${modelPascal}(undefined);
                  setDrawerOpen(true);
                }}
              >
                {t('create')}
              </Button>,
            ]}
            options={{
              density: true,
              fullScreen: true,
              setting: true,
              reload: true,
            }}
            size="middle"
            bordered
            cardBordered={false}
            scroll={{ y: tableScrollY, x: 1000 }}
          />
        </div>
      </ContentContainer>

      {/* ${service.modelName} 编辑/创建 Drawer */}
      <${modelPascal}Drawer
        open={drawerOpen}
        mode={drawerMode}
        data={selected${modelPascal}}
        onClose={() => {
          setDrawerOpen(false);
          setSelected${modelPascal}(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default ${modelPascal}Management;
`
  return code
}

// ==============================
// 编辑抽屉 *Drawer.tsx
// ==============================

/**
 * 根据字段类型推断 ProForm 组件
 */
function getFieldFormComponent(field: ParsedField): {
  component: string
  extraProps: string
} {
  if (field.isBoolean) {
    return { component: 'ProFormSwitch', extraProps: '' }
  }
  if (field.isEnum && field.enumValues) {
    const lower = field.name.toLowerCase()
    if (lower.includes('status')) {
      return {
        component: 'ProFormRadio.Group',
        extraProps: `        options={getStatusOptions(t)}
        fieldProps={{ optionType: 'button', buttonStyle: 'solid' }}`,
      }
    }
    return { component: 'ProFormSelect', extraProps: `        options={${field.name}Options}` }
  }
  if (field.isInteger && field.name.toLowerCase().includes('sort')) {
    return {
      component: 'ProFormDigit',
      extraProps: `        fieldProps={{ precision: 0, min: 0 }}`,
    }
  }
  if (field.isDate) {
    return {
      component: 'ProFormDateTimePicker',
      extraProps: `        fieldProps={{ style: { width: '100%' } }}`,
    }
  }
  if (field.name.toLowerCase().includes('description') || field.name.toLowerCase().includes('remark')) {
    return {
      component: 'ProFormTextArea',
      extraProps: `        fieldProps={{ allowClear: true, rows: 2 }}`,
    }
  }
  return {
    component: 'ProFormText',
    extraProps: `        fieldProps={{ allowClear: true }}`,
  }
}

export function generateDrawerCode(options: ReactPageTemplateOptions): string {
  const {
    service,
    serviceName = 'admin',
  } = options

  const modelPascal = toPascalCase(service.modelName)
  const fileName = service.kebabName.replace(/-/g, '')
  const crudPaths = getCrudPaths(service)
  const hasCreate = !!crudPaths.create
  const hasUpdate = !!crudPaths.update

  // 表单字段（排除系统字段和ID）
  const formFields = service.fields.filter(f => !['id'].includes(f.name) && !f.isDate)

  // 判断是否需要 status 相关组件
  const hasStatusEnum = service.fields.some(f => f.isEnum && f.name.toLowerCase().includes('status'))

  // 收集所有需要 import 的 ProForm 组件
  const formComponents = new Set<string>()
  for (const field of formFields) {
    const { component } = getFieldFormComponent(field)
    formComponents.add(component)
  }

  // 生成表单项代码
  const formItemCodes = formFields.map(f => {
    const form = getFieldFormComponent(f)
    const isRequired = !f.isBoolean && !['sortOrder', 'description', 'remark'].includes(f.name)

    let result = `      <${form.component}
        name="${f.name}"
        label={t('${f.name}')}
        placeholder={t('${f.name}Placeholder')}`

    if (isRequired) {
      result += `\n        rules={[{ required: true, message: t('required${toPascalCase(f.name)}') }]}`
    }

    if (form.extraProps) {
      result += `\n${form.extraProps}`
    }

    result += `\n      />`
    return result
  }).join('\n\n')

  // initialValues
  const defaultValues: string[] = []
  for (const f of formFields) {
    if (f.isBoolean) {
      const defaultTrue = ['isEnabled', 'isDefault'].includes(f.name)
      defaultValues.push(`        ${f.name}: ${defaultTrue},`)
    } else if (f.isInteger && f.name.toLowerCase().includes('sort')) {
      defaultValues.push(`        ${f.name}: 1,`)
    } else if (f.isEnum && f.name.toLowerCase().includes('status')) {
      defaultValues.push(`        ${f.name}: 'ON',`)
    }
  }

  const drawerSize = formFields.length > 8 ? 600 : 480

  let code = `import { useRef, useState } from 'react';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  DrawerForm,
${Array.from(formComponents).map(c => `  ${c},`).join('\n')}
} from '@ant-design/pro-components';
import { App } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { ${fileName}_${modelPascal} as ${modelPascal} } from '@/api/generated/${serviceName}/service/v1';
import { useCreate${modelPascal}, useUpdate${modelPascal} } from '@/api/hooks/${fileName}';
${hasStatusEnum ? `import { getStatusOptions } from '../constants';\n` : ''}
interface ${modelPascal}DrawerProps {
  open: boolean;
  mode: 'create' | 'edit';
  data?: ${modelPascal};
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * ${service.modelName} 编辑/创建抽屉组件
 */
const ${modelPascal}Drawer: React.FC<${modelPascal}DrawerProps> = ({
  open,
  mode,
  data,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('${fileName}');
  const formRef = useRef<ProFormInstance>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const [confirmLoading, setConfirmLoading] = useState(false);
${hasCreate ? `
  const createMutation = useCreate${modelPascal}({
    onSuccess: () => {
      message.success(t('createSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['list${modelPascal}s'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('createFailed'));
    },
  });
` : ''}${hasUpdate ? `
  const updateMutation = useUpdate${modelPascal}({
    onSuccess: () => {
      message.success(t('updateSuccess'));
      onSuccess();
      onClose();
      queryClient.invalidateQueries({ queryKey: ['list${modelPascal}s'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('updateFailed'));
    },
  });
` : ''}
  const handleSubmit = async (values: any) => {
    setConfirmLoading(true);
    try {
${hasCreate ? `      if (mode === 'create') {
        await createMutation.mutateAsync({ data: values });
      }` : ''}${hasUpdate ? `${hasCreate ? ' else' : ''} if (data?.id) {
        await updateMutation.mutateAsync({ id: data.id, values });
      }` : ''}
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
${defaultValues.join('\n')}
            }
      }
      onFinish={handleSubmit}
      submitter={{
        searchConfig: {
          submitText: t('common:button.submit'),
          resetText: t('common:button.cancel'),
        },
        submitButtonProps: {
          loading: confirmLoading${hasCreate ? ' || createMutation.isPending' : ''}${hasUpdate ? ' || updateMutation.isPending' : ''},
        },
        resetButtonProps: {
          onClick: onClose,
        },
      }}
      drawerProps={{
        destroyOnClose: true,
        onClose,
        width: ${drawerSize},
      }}
    >
${formItemCodes}
    </DrawerForm>
  );
};

export default ${modelPascal}Drawer;
`
  return code
}

/**
 * 生成 constants.ts 文件（如果包含 status 枚举）
 */
export function generateConstantsCode(service: ParsedService): string | null {
  const hasStatusEnum = service.fields.some(f => f.isEnum && f.name.toLowerCase().includes('status'))
  if (!hasStatusEnum) return null

  const statusField = service.fields.find(f => f.isEnum && f.name.toLowerCase().includes('status'))
  if (!statusField?.enumValues) return null

  const statusMapEntries = statusField.enumValues.map(v => {
    const label = v === 'ON' ? '启用' : v === 'OFF' ? '禁用' : v
    const color = v === 'ON' ? 'success' : v === 'OFF' ? 'error' : 'default'
    return `    ${v}: { text: t('${label}'), color: '${color}' },`
  }).join('\n')

  const statusOptions = statusField.enumValues.map(v => {
    const label = v === 'ON' ? '启用' : v === 'OFF' ? '禁用' : v
    return `    { label: t('${label}'), value: '${v}' },`
  }).join('\n')

  return `/**
 * ${service.modelName} 模块常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

/** 状态映射 */
export function getStatusMap(t: TFn) {
  return {
${statusMapEntries}
  };
}

/** 状态选项 */
export function getStatusOptions(t: TFn) {
  return [
${statusOptions}
  ];
}
`
}
