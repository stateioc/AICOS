
const mockServer = require('./mock-server');

module.exports = {
  host: process.env.BK_APP_HOST,
  port: process.env.BK_APP_PORT,
  publicPath: process.env.BK_STATIC_URL,
  open: true,
  replaceStatic: true,
  filenameHashing: false,
  outputDir: '../backend/static/dist',

  // webpack config 配置
  configureWebpack() {
    return {
      devServer: {
        setupMiddlewares: mockServer,
      },
    };
  },
};
