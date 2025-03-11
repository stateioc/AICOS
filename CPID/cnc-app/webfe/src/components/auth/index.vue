<script>
export default {
  data() {
    return {
      isShow: false,
      loginWindow: null,
      checkWindowTimer: -1,
    };
  },
  created() {
    window.addEventListener('message', this.messageListener, false);
  },
  beforeDestroy() {
    window.removeEventListener('message', this.messageListener, false);
  },
  methods: {
    showLoginModal(data = {}) {
      if (this.isShow) return;
      this.isShow = true;
      let url = data.login_url;
      if (!url) {
        // const callbackUrl = `${location.origin}/static/login_success.html?is_ajax=1`;
        const callbackUrl = window.location.href;
        const appCode = window.BKPAAS_APP_ID;
        const loginUrl = window.BK_LOGIN_URL;
        url = `${loginUrl}?app_id=${appCode}&c_url=${callbackUrl}`;
      }
      const width = 700;
      const height = 510;
      const { availHeight, availWidth } = window.screen;
      this.loginWindow = window.open(url, '_blank', `
        width=${width},
        height=${height},
        left=${(availWidth - width) / 2},
        top=${(availHeight - height) / 2},
        channelmode=0,
        directories=0,
        fullscreen=0,
        location=0,
        menubar=0,
        resizable=0,
        scrollbars=0,
        status=0,
        titlebar=0,
        toolbar=0,
        close=0
      `);
      this.checkWinClose();
    },
    checkWinClose() {
      this.checkWindowTimer && clearTimeout(this.checkWindowTimer);
      this.checkWindowTimer = setTimeout(() => {
        if (!this.loginWindow || this.loginWindow.closed) {
          this.hideLoginModal();
          clearTimeout(this.checkWindowTimer);
          return;
        }
        this.checkWinClose();
      }, 300);
    },
    messageListener({ data = {} }) {
      if (data === null || typeof data !== 'object' || data.target !== 'bk-login' || !this.loginWindow) return;

      this.hideLoginModal();
    },
    hideLoginModal() {
      this.isShow = false;
      if (this.loginWindow) {
        this.loginWindow.close();
      }
      this.loginWindow = null;
    },
  },
  render() {
    return '';
  },
};
</script>
