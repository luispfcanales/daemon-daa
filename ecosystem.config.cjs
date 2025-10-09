module.exports = {
  apps: [{
    name: "daemon-dns",
    script: "./bin/main.exe",
    watch: false,
    instances: 1,
    exec_mode: "fork",
    env: {
      NODE_ENV: "production",
      PORT: 3000
    }
  }]
}