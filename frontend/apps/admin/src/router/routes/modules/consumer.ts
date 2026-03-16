import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const consumerRoutes: RouteRecordRaw[] = [
  {
    name: 'Consumer',
    path: '/consumer',
    component: BasicLayout,
    meta: {
      icon: 'lucide:users',
      order: 10,
      title: 'C端用户管理',
      authority: ['consumer:view'],
    },
    children: [
      // 认证模块
      {
        name: 'ConsumerAuth',
        path: '/consumer/auth',
        meta: {
          icon: 'lucide:shield-check',
          title: '用户认证',
          authority: ['consumer:auth:view'],
        },
        children: [
          {
            name: 'ConsumerRegister',
            path: '/consumer/auth/register',
            component: () => import('#/views/consumer/auth/register.vue'),
            meta: {
              icon: 'lucide:user-plus',
              title: '用户注册',
              authority: ['consumer:auth:register'],
              ignoreAccess: true, // 注册页面不需要登录
            },
          },
          {
            name: 'ConsumerLogin',
            path: '/consumer/auth/login',
            component: () => import('#/views/consumer/auth/login.vue'),
            meta: {
              icon: 'lucide:log-in',
              title: '用户登录',
              authority: ['consumer:auth:login'],
              ignoreAccess: true, // 登录页面不需要登录
            },
          },
        ],
      },

      // 用户信息模块
      {
        name: 'ConsumerUser',
        path: '/consumer/user',
        meta: {
          icon: 'lucide:user',
          title: '用户信息',
          authority: ['consumer:user:view'],
        },
        children: [
          {
            name: 'ConsumerProfile',
            path: '/consumer/user/profile',
            component: () => import('#/views/consumer/user/profile.vue'),
            meta: {
              icon: 'lucide:user-circle',
              title: '个人信息',
              authority: ['consumer:user:profile'],
            },
          },
          {
            name: 'ConsumerLoginLogs',
            path: '/consumer/user/login-logs',
            component: () => import('#/views/consumer/user/login-logs.vue'),
            meta: {
              icon: 'lucide:history',
              title: '登录日志',
              authority: ['consumer:user:logs'],
            },
          },
        ],
      },

      // 财务模块
      {
        name: 'ConsumerFinance',
        path: '/consumer/finance',
        meta: {
          icon: 'lucide:wallet',
          title: '财务管理',
          authority: ['consumer:finance:view'],
        },
        children: [
          {
            name: 'ConsumerFinanceAccount',
            path: '/consumer/finance/account',
            component: () => import('#/views/consumer/finance/account.vue'),
            meta: {
              icon: 'lucide:credit-card',
              title: '账户余额',
              authority: ['consumer:finance:account'],
            },
          },
          {
            name: 'ConsumerFinanceRecharge',
            path: '/consumer/finance/recharge',
            component: () => import('#/views/consumer/finance/recharge.vue'),
            meta: {
              icon: 'lucide:arrow-down-to-line',
              title: '充值',
              authority: ['consumer:finance:recharge'],
            },
          },
          {
            name: 'ConsumerFinanceWithdraw',
            path: '/consumer/finance/withdraw',
            component: () => import('#/views/consumer/finance/withdraw.vue'),
            meta: {
              icon: 'lucide:arrow-up-from-line',
              title: '提现',
              authority: ['consumer:finance:withdraw'],
            },
          },
          {
            name: 'ConsumerFinanceTransactions',
            path: '/consumer/finance/transactions',
            component: () =>
              import('#/views/consumer/finance/transactions.vue'),
            meta: {
              icon: 'lucide:receipt',
              title: '财务流水',
              authority: ['consumer:finance:transactions'],
            },
          },
        ],
      },

      // 媒体模块
      {
        name: 'ConsumerMedia',
        path: '/consumer/media',
        meta: {
          icon: 'lucide:image',
          title: '媒体管理',
          authority: ['consumer:media:view'],
        },
        children: [
          {
            name: 'ConsumerMediaUpload',
            path: '/consumer/media/upload',
            component: () => import('#/views/consumer/media/upload.vue'),
            meta: {
              icon: 'lucide:upload',
              title: '上传文件',
              authority: ['consumer:media:upload'],
            },
          },
          {
            name: 'ConsumerMediaList',
            path: '/consumer/media/list',
            component: () => import('#/views/consumer/media/list.vue'),
            meta: {
              icon: 'lucide:folder-open',
              title: '文件列表',
              authority: ['consumer:media:list'],
            },
          },
        ],
      },

      // 物流模块
      {
        name: 'ConsumerLogistics',
        path: '/consumer/logistics',
        meta: {
          icon: 'lucide:truck',
          title: '物流管理',
          authority: ['consumer:logistics:view'],
        },
        children: [
          {
            name: 'ConsumerLogisticsQuery',
            path: '/consumer/logistics/query',
            component: () => import('#/views/consumer/logistics/query.vue'),
            meta: {
              icon: 'lucide:search',
              title: '物流查询',
              authority: ['consumer:logistics:query'],
            },
          },
          {
            name: 'ConsumerLogisticsTracking',
            path: '/consumer/logistics/tracking',
            component: () => import('#/views/consumer/logistics/tracking.vue'),
            meta: {
              icon: 'lucide:map-pin',
              title: '物流轨迹',
              authority: ['consumer:logistics:tracking'],
            },
          },
        ],
      },

      // 运费计算模块
      {
        name: 'ConsumerFreight',
        path: '/consumer/freight',
        meta: {
          icon: 'lucide:calculator',
          title: '运费计算',
          authority: ['consumer:freight:view'],
        },
        children: [
          {
            name: 'ConsumerFreightCalculator',
            path: '/consumer/freight/calculator',
            component: () => import('#/views/consumer/freight/calculator.vue'),
            meta: {
              icon: 'lucide:calculator',
              title: '运费计算器',
              authority: ['consumer:freight:calculate'],
            },
          },
          {
            name: 'ConsumerFreightTemplates',
            path: '/consumer/freight/templates',
            component: () => import('#/views/consumer/freight/templates.vue'),
            meta: {
              icon: 'lucide:file-text',
              title: '运费模板',
              authority: ['consumer:freight:templates'],
            },
          },
        ],
      },
    ],
  },
];

export default consumerRoutes;
