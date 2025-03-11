<template>
  <div id="app" :class="systemCls">
    <bk-navigation
      :header-title="nav.id"
      :side-title="nav.title"
      :navigation-type="curNav.nav"
      :need-menu="curNav.needMenu"
      :default-open="true"
      @toggle="handleToggle">
      <template slot="header">
        <div class="navigation-header">
          <div class="header-title">
            <span class="ml5">{{nav.id}}</span>
          </div>
          <bk-popover theme="light navigation-profile" :arrow="false" offset="-20, 10" placement="bottom-start">
            <div class="header-user" :class="{ 'is-left': curNav.nav === 'left-right' }">
              {{user.username}}
              <i class="bk-icon icon-down-shape"></i>
            </div>
            <template slot="content">
              <ul class="navigation-admin">
                <li class="nav-item" v-for="userItem in admin.list" :key="userItem">
                  {{userItem}}
                </li>
              </ul>
            </template>
          </bk-popover>
        </div>
      </template>
      <template slot="menu">
        <bk-navigation-menu
          ref="menu"
          @select="handleSelect"
          :default-active="nav.id"
          :before-nav-change="beforeNavChange"
          :toggle-active="nav.toggle">
          <bk-navigation-menu-item
            v-for="item in nav.list"
            :key="item.name"
            :has-child="item.children && !!item.children.length"
            :group="item.group"
            :icon="item.icon"
            :disabled="item.disabled"
            :url="item.url"
            :id="item.name">
            <span>{{item.name}}</span>
            <div slot="child">
              <bk-navigation-menu-item
                :key="child.name"
                v-for="child in item.children"
                :id="child.name"
                :disabled="child.disabled"
                :icon="child.icon"
                :default-active="child.active">
                <span>{{child.name}}</span>
              </bk-navigation-menu-item>
            </div>
          </bk-navigation-menu-item>
        </bk-navigation-menu>
      </template>
      <div class="navigation-content p20">
        <main class="main-content" v-bkloading="{ isLoading: mainContentLoading, opacity: 0 }">
          <router-view :key="routerKey" v-show="!mainContentLoading" />
        </main>
      </div>
      <template slot="footer">
        <div class="navigation-footer">
          Copyright © 2012-{{curYear}} Tencent BlueKing. All Rights Reserved. 腾讯蓝鲸 版权所有
        </div>
      </template>
    </bk-navigation>
    <app-auth ref="bkAuth"></app-auth>
  </div>
</template>
<script>
import { mapGetters } from 'vuex';
import { bus } from '@/common/bus';

export default {
  name: 'App',
  data() {
    return {
      routerKey: +new Date(),
      systemCls: 'mac',
      nav: {
        list: [
          {
            name: '总览',
            icon: 'icon-tree-application-shape',
            url: 'overview',
          },
          {
            name: '开通资源',
            icon: 'icon-tree-group-shape',
            url: 'resource',
          },
          {
            name: '操作记录',
            icon: 'icon-tree-group-shape',
            url: 'record',
          },
        ],
        id: '首页',
        toggle: false,
        submenuActive: false,
        title: '算力互联互通平台',
      },
      admin: {
        list: [
          '项目管理',
          '权限中心',
          '退出',
        ],
      },
    };
  },
  computed: {
    ...mapGetters(['mainContentLoading', 'user']),
    curNav() {
      return {
        nav: 'left-right',
        needMenu: true,
        name: '左右结构导航',
      };
    },
    curYear() {
      return (new Date()).getFullYear();
    },
  },
  watch: {
    '$route'() {
      this.nav.id = this.$route.meta ? this.$route.meta.matchRoute : this.$route.name;
    },
  },
  created() {
    const platform = window.navigator.platform.toLowerCase();
    if (platform.indexOf('win') === 0) {
      this.systemCls = 'win';
    }
  },
  mounted() {
    bus.$on('show-login-modal', (data) => {
      this.$refs.bkAuth.showLoginModal(data);
    });
    bus.$on('close-login-modal', () => {
      this.$refs.bkAuth.hideLoginModal();
      setTimeout(() => {
        window.location.reload();
      }, 0);
    });
  },
  methods: {
    goPage(idx) {
      if (idx) {
        this.$router.push({
          name: idx,
        });
      }
    },
    handleSelect(id, item) {
      this.nav.url = item.url;
      this.nav.id = id;
      if (item.url) {
        this.goPage(item.url);
      }
    },
    handleToggle(v) {
      this.nav.toggle = v;
    },
    beforeNavChange(newId, oldId) {
      console.info(newId, oldId);
      return true;
    },
  },
};
</script>

<style lang="postcss">
    @import './css/reset.css';
    @import './css/app.css';

    .main-content {
        min-height: 300px;
    }

    .bk-navigation {
        outline: 1px solid #ebebeb;
        .bk-navigation-wrapper {
            height: calc(100vh - 252px)!important;
        }
    }

    .navigation-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 1;
        height: 100%;
        font-size: 14px;
        .header-nav {
            display: flex;
            padding: 0;
            margin: 0;
            &-item {
                list-style: none;
                margin-right: 40px;
                color: #96A2B9;
                &.item-active {
                    color: #FFFFFF !important;
                }
                &:hover {
                    cursor: pointer;
                    color: #D3D9E4;
                }
            }
        }
        .header-title {
            color: #63656E;
            font-size: 16px;
            display: flex;
            align-items: center;
            margin-left: -6px;
        }
        .header-user {
            height: 100%;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #96A2B9;
            .bk-icon {
                margin-left: 5px;
                font-size: 12px;
            }
            &.is-left {
                @mixin is-left-mixin false;
            }
            &:hover {
                cursor: pointer;
                color: #D3D9E4;
            }
        }
    }

    .navigation-content {
        min-height: calc(100% - 84px);
        background: #FFFFFF;
        box-shadow: 0px 2px 4px 0px rgba(25,25,41,0.05);
        border-radius: 2px;
        border: 1px solid rgba(220,222,229,1);
    }

    .navigation-footer {
        height: 52px;
        width: 100%;
        margin: 32px 0 0 ;
        display: flex;
        align-items: center;
        justify-content: center;
        border-top: 1px solid #DCDEE5;
        color: #63656E;
        font-size: 12px;
    }

    .navigation-admin {
        @mixin popover-panel-mxin 170px #63656E;
    }

    .tippy-popper {
        .tippy-tooltip.navigation-profile-theme {
            padding: 0;
            border-radius: 0;
            box-shadow: none;
        }
    }
</style>
