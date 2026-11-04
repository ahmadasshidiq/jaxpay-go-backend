const path = require("path");

// Root project: .env dan keys/ dibaca relatif ke folder ini
const ROOT = __dirname;
const BIN = path.join(ROOT, "bin");

const common = {
  cwd: ROOT,
  interpreter: "none", // binary Go, bukan script Node
  autorestart: true,
  watch: false,
  max_memory_restart: "500M",
  env: {
    APP_ENV: "production",
  },
};

module.exports = {
  apps: [
    {
      ...common,
      name: "jaxpay-gateway",
      script: path.join(BIN, "jaxpay-gateway"),
    },
    {
      ...common,
      name: "jaxpay-worker",
      script: path.join(BIN, "jaxpay-worker"),
    },
  ],
};
