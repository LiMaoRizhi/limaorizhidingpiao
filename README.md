# 狸猫日志客运订票系统 (Limaorizhi Ticketing System)

[![License](https://img.shields.io/badge/license-Copyright-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.5+-4FC08D?logo=vue.js)](https://vuejs.org/)
[![WeChat](https://img.shields.io/badge/WeChat-MiniProgram-07C160?logo=wechat)](https://developers.weixin.qq.com/miniprogram/)

## 📖 项目简介

狸猫日志是一个完整的**客运订票系统**,提供从乘客购票、司机调度到后台管理的完整解决方案。系统采用前后端分离架构,支持微信小程序端和管理后台,具备实时车辆追踪、在线支付、订单管理等核心功能。

## ✨ 核心功能

### 🚀 微信小程序端
- **在线购票**: 线路查询、班次选择、在线支付
- **订单管理**: 订单列表、详情查看、退改签
- **实名认证**: 乘客信息管理与身份验证
- **实时追踪**: 车辆位置实时显示、行程进度监控
- **扫码验票**: 电子票据二维码验证
- **货运服务**: 货物托运下单与管理
- **惊喜礼包**: 营销活动与优惠券系统
- **个人中心**: 个人信息、乘车记录、积分管理

### 💻 管理后台
- **数据看板**: 销售统计、客流分析、运营数据可视化
- **线路管理**: 班次调度、站点配置、票价设置
- **订单管理**: 订单审核、退票处理、异常订单处理
- **用户管理**: 乘客信息管理、司机管理、权限控制
- **车辆管理**: 车辆信息、实时定位、运行状态监控
- **财务管理**: 收入统计、对账管理、财务报表
- **营销管理**: 优惠券发放、活动策划、礼包管理

### 🔧 后端服务
- **RESTful API**: 基于Gin框架的高性能API服务
- **身份认证**: JWT Token认证机制
- **微信支付**: 集成微信商户支付API v3
- **实时定位**: WebSocket实时位置推送
- **数据库**: MySQL + GORM ORM
- **二维码生成**: 电子票据二维码
- **文件上传**: 图片、证件等材料上传

## 🛠️ 技术栈

### 前端管理后台
- **框架**: Vue 3.5+ (Composition API)
- **语言**: TypeScript 5.6+
- **UI组件**: Element Plus 2.9+
- **构建工具**: Vite 5.4+
- **状态管理**: Pinia 2.2+
- **路由**: Vue Router 4.4+
- **HTTP客户端**: Axios 1.7+
- **地图**: Leaflet 1.9+
- **图表**: ECharts 5.5+
- **动画**: GSAP 3.15+

### 后端服务
- **语言**: Go 1.26+
- **Web框架**: Gin 1.12+
- **ORM**: GORM 1.31+
- **数据库**: MySQL 8.0+
- **认证**: JWT (golang-jwt)
- **配置**: YAML
- **二维码**: go-qrcode

### 微信小程序
- **框架**: 微信原生小程序
- **地图**: 腾讯地图SDK
- **支付**: 微信支付API v3
- **定位**: 微信位置API

## 📦 项目结构

```
limaorizhidingpiao/
├── limaorizhi-dingpiaoWX/          # 微信小程序端
│   ├── pages/                      # 页面组件
│   ├── components/                 # 公共组件
│   ├── utils/                      # 工具函数
│   ├── images/                     # 静态资源
│   └── app.json                    # 小程序配置
│
├── limaorizhi-dingpiaoHD/          # 管理后台前端
│   └── limaorizhi-admin/
│       ├── src/
│       │   ├── views/              # 页面视图
│       │   ├── components/         # 组件
│       │   ├── api/                # API接口
│       │   ├── stores/             # 状态管理
│       │   ├── router/             # 路由配置
│       │   └── utils/              # 工具函数
│       ├── package.json
│       └── vite.config.ts
│
├── limaorizhi-dingpiaoHD/          # 后端服务
│   └── limaorizhi-server/
│       ├── cmd/server/             # 主入口
│       ├── internal/
│       │   ├── handler/            # HTTP处理器
│       │   ├── service/            # 业务逻辑
│       │   ├── model/              # 数据模型
│       │   ├── middleware/         # 中间件
│       │   ├── router/             # 路由配置
│       │   └── pkg/                # 工具包
│       ├── migrations/             # 数据库迁移
│       ├── config.yaml             # 配置文件
│       └── go.mod
│
└── limaorizhi-server/              # 生产环境部署文件
    ├── config.yaml                 # 生产配置
    ├── .env                        # 环境变量
    ├── start.sh                    # 启动脚本
    ├── stop.sh                     # 停止脚本
    └── nginx_limaorizhi.conf       # Nginx配置
```

## 🚀 快速开始

### 前置要求
- Node.js >= 18.x
- Go >= 1.26
- MySQL >= 8.0
- 微信开发者工具

### 1. 克隆项目
```bash
git clone https://github.com/your-username/limaorizhi-ticketing.git
cd limaorizhi-ticketing
```

### 2. 后端服务

```bash
cd limaorizhi-dingpiaoHD/limaorizhi-server

# 安装依赖
go mod download

# 配置数据库
# 编辑 config.yaml,设置MySQL连接信息

# 初始化数据库
mysql -u root -p < migrations/schema.sql

# 运行服务
go run cmd/server/main.go
```

### 3. 管理后台前端

```bash
cd limaorizhi-dingpiaoHD/limaorizhi-admin

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

### 4. 微信小程序

1. 使用微信开发者工具打开 `limaorizhi-dingpiaoWX` 目录
2. 修改 `config.js` 中的API地址
3. 编译并预览

## ⚙️ 配置说明

### 后端配置 (config.yaml)
```yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  dbname: limaorizhi

jwt:
  secret: your-secret-key
  expire: 24h

wechat:
  appid: your-appid
  secret: your-secret
  mch_id: your-merchant-id
  api_v3_key: your-api-v3-key
```

### 前端配置 (.env.development)
```
VITE_API_BASE_URL=http://localhost:8080/api
```

## 🌐 部署指南

### Linux服务器部署

1. **编译后端**
```bash
GOOS=linux GOARCH=amd64 go build -o limaorizhi-server ./cmd/server
```

2. **上传文件到服务器**
```bash
scp limaorizhi-server user@server:/opt/limaorizhi/
scp -r limaorizhi-server/* user@server:/opt/limaorizhi/
```

3. **配置Nginx**
```bash
sudo cp nginx_limaorizhi.conf /etc/nginx/conf.d/
sudo nginx -t
sudo systemctl reload nginx
```

4. **启动服务**
```bash
chmod +x start.sh
./start.sh
```

## 📱 功能截图

*(建议添加以下截图)*
- 微信小程序首页
- 购票流程
- 车辆实时追踪
- 管理后台数据看板
- 订单管理界面

## 🔐 安全说明

- 敏感配置文件(`.env`, `cert/`)已加入 `.gitignore`
- 生产环境请修改默认JWT密钥
- 定期更新SSL证书
- 建议使用HTTPS协议

## 📄 许可证

Copyright © 2024 狸猫日志. All rights reserved.

本项目为商业软件,未经授权不得用于商业用途。

## 👥 贡献指南

欢迎提交Issue和Pull Request!

1. Fork本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启Pull Request

## 📞 联系方式

如有问题或建议,请通过以下方式联系:
- Issue: [GitHub Issues](https://github.com/your-username/limaorizhi-ticketing/issues)
- Email: your-email@example.com

## 🙏 致谢

感谢以下开源项目:
- [Vue.js](https://vuejs.org/)
- [Element Plus](https://element-plus.org/)
- [Gin](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [Leaflet](https://leafletjs.com/)
- [ECharts](https://echarts.apache.org/)

---

**⭐ 如果这个项目对你有帮助,请给个Star!**
