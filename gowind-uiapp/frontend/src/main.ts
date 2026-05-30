import {createApp} from 'vue'
import Antd from 'ant-design-vue';
import {DatePicker, Tabs, message} from 'ant-design-vue';

import App from './App.vue'

import 'ant-design-vue/dist/reset.css';
import './style.css';

import VxeUIBase from 'vxe-pc-ui'
import 'vxe-pc-ui/es/style.css'

import VxeUITable from 'vxe-table'
import 'vxe-table/es/style.css'

import i18n from './i18n'

const app = createApp(App);

app.use(i18n)
app.use(Antd).use(DatePicker).use(Tabs);
app.use(VxeUIBase).use(VxeUITable);

app.mount('#app')

app.config.globalProperties.$message = message;
