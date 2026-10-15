export const layouts = { mobile: { width: 390, height: 844 }, desktop: { width: 1440, height: 1080 } };
export const pages = [
  { id: 'dashboard', title: 'Dashboard', group: '运行总览', description: '查看域名、证书、反代和代理入口数量，控制 Nginx 与 sing-box。', features: ['运行状态与监听地址', '进程启动、停止、重启', '日志、配置与调试开关'], entry: '登录后首页', captures: [['dashboard', '运行总览'], ['dashboard-logs', 'sing-box 日志'], ['dashboard-config', 'Nginx 配置']] },
  { id: 'domains', title: '域名管理', group: '资源管理', description: '统一维护域名、DNS 检查与证书生命周期。', features: ['新增域名', 'DNS 检查与管理域名', '申请、续期和删除证书'], entry: '左侧导航 → 域名管理', captures: [['domains', '域名列表'], ['domains-create', '新增域名'], ['domains-dns', 'DNS 检查结果']] },
  { id: 'reverse', title: '反向代理', group: '资源管理', description: '把域名安全转发到本机或内网服务。', features: ['HTTP/HTTPS 目标', 'WebSocket、真实 IP 和 Preserve Host', '编辑与删除规则'], entry: '左侧导航 → 反向代理', captures: [['reverse', '规则列表'], ['reverse-edit', '编辑规则']] },
  { id: 'vless', title: '代理入口', group: '资源管理', description: '管理 VLESS Reality Vision、AnyTLS 与 Hysteria2 入站。', features: ['创建与编辑入站', '启用、停用和删除', '查看配置与分享二维码'], entry: '左侧导航 → 代理入口', captures: [['vless', '入口列表'], ['vless-create', '创建入口'], ['vless-config', '入口配置'], ['vless-share', '分享二维码']] },
  { id: 'settings', title: '系统设置', group: '系统', description: '查看 ACME、路径、版本和完整 Runtime 状态。', features: ['系统信息与 Runtime JSON', '关闭初始管理端口'], entry: '左侧导航 → 系统设置', captures: [['settings', '系统信息'], ['settings-runtime', 'Runtime JSON']] },
  { id: 'audit', title: '审计日志', group: '系统', description: '追踪登录、证书、域名、反代、入站和 Runtime 操作。', features: ['按时间查看操作记录', '展开详情 JSON'], entry: '左侧导航 → 审计日志', captures: [['audit', '操作日志'], ['audit-detail', '日志详情']] },
  { id: 'login', title: '登录', group: '账号', description: '使用管理密码进入 Proxy-Go。', features: ['密码输入', '登录失败反馈'], entry: '登出后返回', captures: [['login', '登录页']] },
  { id: 'mock-debug', title: 'Mock 调试面板', group: '开发工具', description: '在浏览器中控制 Mock API 行为并检查请求。', features: ['模拟响应延迟和 503 故障', '切换登录状态与重置数据', '查看请求记录'], entry: '右下角 Mock 调试按钮', captures: [['mock-debug', '请求记录与故障注入']] },
];
