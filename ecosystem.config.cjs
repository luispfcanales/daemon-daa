module.exports = {
  apps: [{
    name: "daemon-dns",
    script: "./main.exe",
    watch: false,
    instances: 1,
    exec_mode: "fork",
    env: {
      NODE_ENV: "production",
      PORT: 3000,
      PATH: process.env.PATH,
      PSModulePath: process.env.PSModulePath,
      SystemRoot: process.env.SystemRoot
    }
  }]
}