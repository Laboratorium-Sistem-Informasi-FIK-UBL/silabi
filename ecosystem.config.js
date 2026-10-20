module.exports = {
  apps: [
    {
      name: "silabi-web",
      script: "./bin/silabi-web",
      interpreter: "none",
      cwd: __dirname,
      env_file: ".env",
      autorestart: true,
      watch: false,
      max_memory_restart: "200M",
    },
  ],
};
