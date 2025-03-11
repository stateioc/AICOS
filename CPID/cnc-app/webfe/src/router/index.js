/**
 * @file router 配置
 * @author
 */

import Vue from 'vue';
import VueRouter from 'vue-router';

import store from '@/store';
import http from '@/api';
import preload from '@/common/preload';

const MainEntry = () => import(/* webpackChunkName: 'entry' */'@/views');
// 总览
const OverView = () => import(/* webpackChunkName: 'example1' */'@/views/example1');
// 开通资源
const Resource = () => import(/* webpackChunkName: 'example2' */'@/views/resource');
// 操作记录
const RecordList = () => import(/* webpackChunkName: 'example2' */'@/views/record-list/index');
// import Example1 from '@/views/example1'
const Example2 = () => import(/* webpackChunkName: 'example2' */'@/views/example2');
// import Example2 from '@/views/example2'
const NotFound = () => import(/* webpackChunkName: 'none' */'@/views/404');
// import NotFound from '@/views/404'

Vue.use(VueRouter);

const routes = [
  {
    path: window.SITE_URL,
    name: 'appMain',
    component: MainEntry,
    alias: '',
    children: [
      {
        path: 'overview',
        alias: '',
        name: 'overview',
        component: OverView,
        meta: {
          matchRoute: '总览',
        },
      },
      {
        path: 'example2',
        name: 'example2',
        component: Example2,
        meta: {
          matchRoute: '登陆信息',
        },
      },
      {
        path: 'resource',
        name: 'resource',
        component: Resource,
        meta: {
          matchRoute: '开通资源',
        },
      },
      {
        path: 'record',
        name: 'record',
        component: RecordList,
        meta: {
          matchRoute: '操作记录',
        },
      },
    ],
  },
  // 404
  {
    path: '*',
    name: '404',
    component: NotFound,
  },
];

const router = new VueRouter({
  mode: 'history',
  routes,
});

const cancelRequest = async () => {
  const allRequest = http.queue.get();
  const requestQueue = allRequest.filter(request => request.cancelWhenRouteChange);
  await http.cancel(requestQueue.map(request => request.requestId));
};

let preloading = true;
let canceling = true;
let pageMethodExecuting = true;

router.beforeEach(async (to, from, next) => {
  canceling = true;
  await cancelRequest();
  canceling = false;
  next();
});

router.afterEach(async (to) => {
  store.commit('setMainContentLoading', true);

  preloading = true;
  await preload();
  preloading = false;

  const pageDataMethods = [];
  const routerList = to.matched;
  routerList.forEach((r) => {
    Object.values(r.instances).forEach((vm) => {
      if (typeof vm.fetchPageData === 'function') {
        pageDataMethods.push(vm.fetchPageData());
      }
      if (typeof vm.$options.preload === 'function') {
        pageDataMethods.push(vm.$options.preload.call(vm));
      }
    });
  });

  pageMethodExecuting = true;
  await Promise.all(pageDataMethods);
  pageMethodExecuting = false;

  if (!preloading && !canceling && !pageMethodExecuting) {
    store.commit('setMainContentLoading', false);
  }
});

export default router;
