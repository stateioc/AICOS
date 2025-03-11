<template>
  <div class="example2">
    <bk-card title="当前登陆人信息">
      <div class="user_name">用户名： {{$store.getters.user.username}}</div>
      <div class="avatar_url">头像： <img :src="$store.getters.user.avatar_url" alt=""></div>
    </bk-card>
  </div>
</template>

<script>
export default {
  components: {},
  data() {
    return {
      initDateTime: new Date(),
    };
  },
  created() {
  },
  methods: {
  },
};
</script>

<style scoped>
    @import './index.css';
</style>
