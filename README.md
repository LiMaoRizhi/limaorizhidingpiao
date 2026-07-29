# 狸猫日志 - 客运订票系统

一个完整的客运售票解决方案,包含微信小程序、管理后台和API服务。

## 这是什么?

狸猫日志是一套客运订票系统,主要解决长途客运的在线售票问题。乘客可以通过微信小程序买票、查看车辆位置、管理订单;管理员可以在后台管理线路、班次、订单和财务数据。

简单说就是:
- 乘客端:微信小程序买票、查订单、看车到哪了
- 管理端:Web后台管理所有业务
- 服务端:Go写的API,处理所有业务逻辑

## 界面预览

### 微信小程序

<table>
  <tr>
    <td><img src="screenshots/screenshot-2.png" width="200"/></td>
    <td><img src="screenshots/screenshot-6.png" width="200"/></td>
    <td><img src="screenshots/screenshot-3.png" width="200"/></td>
  </tr>
  <tr>
    <td align="center">首页</td>
    <td align="center">选座购票</td>
    <td align="center">订单列表</td>
  </tr>
</table>

<table>
  <tr>
    <td><img src="screenshots/screenshot-16.png" width="200"/></td>
    <td><img src="screenshots/screenshot-5.png" width="200"/></td>
    <td><img src="screenshots/screenshot-7.png" width="200"/></td>
  </tr>
  <tr>
    <td align="center">车辆追踪</td>
    <td align="center">个人中心</td>
    <td align="center">司机端</td>
  </tr>
</table>

### 管理后台

<table>
  <tr>
    <td><img src="screenshots/screenshot-13.png" width="400"/></td>
    <td><img src="screenshots/screenshot-11.png" width="400"/></td>
  </tr>
  <tr>
    <td align="center">登录页面</td>
    <td align="center">数据看板</td>
  </tr>
</table>

<table>
  <tr>
    <td><img src="screenshots/screenshot-14.png" width="400"/></td>
    <td><img src="screenshots/screenshot-15.png" width="400"/></td>
  </tr>
  <tr>
    <td align="center">订单管理</td>
    <td align="center">系统设置</td>
  </tr>
</table>

### 3D效果展示

<img src="screenshots/screenshot-10.png" width="600"/>

*电子票3D旋转展示效果*

---

## 主要功能

### 微信小程序(乘客用)

- 查线路、选班次、在线买票
- 查看我的订单,支持退改签
- 实名认证,管理乘车人信息
- 实时看车到哪了,还有多久到站
- 扫码验票上车
- 货物托运(小件快递)
- 领取优惠券和礼包
- 查看乘车记录和个人信息

### 管理后台(管理员用)

- 数据看板:今天卖了多少票、收入多少、哪些线路热门
- 线路管理:添加线路、排班、设置票价
- 订单管理:审核订单、处理退票
- 用户管理:乘客信息、司机信息、权限设置
- 车辆管理:车辆档案、实时位置监控
- 财务统计:对账、报表导出
- 营销活动:发优惠券、搞活动

### 后端API

- RESTful API,用Gin框架写的
- JWT认证,保证接口安全
- 微信支付集成(API v3)
- WebSocket推送车辆实时位置
- MySQL数据库,GORM操作
- 生成电子票二维码
- 文件上传(身份证、照片等)

## 技术选型

### 前端(管理后台)
- Vue 3 + TypeScript
- Element Plus UI组件库
- Vite构建
- Pinia状态管理
- Leaflet地图
- ECharts图表

### 后端
- Go 1.26
- Gin Web框架
- GORM (MySQL ORM)
- JWT认证
- go-qrcode生成二维码

### 小程序
- 微信原生小程序
- 腾讯地图SDK
- 微信支付API v3

## 项目结构

```
limaorizhidingpiao/
├── limaorizhi-dingpiaoWX/          # 微信小程序
│   ├── pages/                      # 页面
│   ├── components/                 # 组件
│   ├── utils/                      # 工具函数
│   └── images/                     # 图片资源
│
├── limaorizhi-dingpiaoHD/
│   ├── limaorizhi-admin/           # 管理后台前端(Vue3)
│   │   ├── src/
│   │   │   ├── views/              # 页面
│   │   │   ├── components/         # 组件
│   │   │   ├── api/                # API请求
│   │   │   └── stores/             # 状态管理
│   │   └── package.json
│   │
│   └── limaorizhi-server/          # 后端API(Go)
│       ├── cmd/server/             # 入口
│       ├── internal/
│       │   ├── handler/            # 控制器
│       │   ├── service/            # 业务逻辑
│       │   ├── model/              # 数据模型
│       │   └── middleware/         # 中间件
│       ├── migrations/             # SQL脚本
│       └── config.yaml             # 配置
│
└── limaorizhi-server/              # 部署相关文件
    ├── config.yaml                 # 生产配置
    ├── start.sh / stop.sh          # 启停脚本
    └── nginx_limaorizhi.conf       # Nginx配置
```

## 怎么跑起来

### 需要准备

- Node.js 18+
- Go 1.26+
- MySQL 8.0+
- 微信开发者工具

### 1. 克隆代码

```bash
git clone https://github.com/LiMaoRizhi/-.git
cd limaorizhidingpiao
```

### 2. 启动后端

```bash
cd limaorizhi-dingpiaoHD/limaorizhi-server

# 下载依赖
go mod download

# 修改 config.yaml,配置数据库连接

# 导入数据库
mysql -u root -p < migrations/schema.sql

# 启动服务
go run cmd/server/main.go
```

### 3. 启动管理后台

```bash
cd limaorizhi-dingpiaoHD/limaorizhi-admin

# 安装依赖
npm install

# 开发模式
npm run dev

# 访问 http://localhost:5173
```

### 4. 打开小程序

用微信开发者工具打开 `limaorizhi-dingpiaoWX` 目录,修改 `config.js` 里的API地址,然后编译预览。

## 配置说明

### 后端配置 (config.yaml)

主要改这几个地方:

```yaml
database:
  host: localhost      # 数据库地址
  port: 3306
  user: root           # 数据库用户名
  password: xxx        # 数据库密码
  dbname: limaorizhi   # 数据库名

jwt:
  secret: 换个复杂的字符串  # JWT密钥,生产环境一定要改

wechat:
  appid: 你的小程序appid
  secret: 你的小程序secret
  mch_id: 微信商户号
  api_v3_key: 微信支付API密钥
```

### 前端配置

在 `limaorizhi-admin/.env.development` 里改API地址:

```
VITE_API_BASE_URL=http://localhost:8080/api
```

## 部署到服务器

### 编译后端

```bash
# 编译成Linux版本
GOOS=linux GOARCH=amd64 go build -o limaorizhi-server ./cmd/server
```

### 上传到服务器

```bash
# 用scp或者FTP上传整个 limaorizhi-server 目录到服务器
scp -r limaorizhi-server/* user@your-server:/opt/limaorizhi/
```

### 配置Nginx

```bash
sudo cp nginx_limaorizhi.conf /etc/nginx/conf.d/
sudo nginx -t
sudo systemctl reload nginx
```

### 启动服务

```bash
chmod +x start.sh
./start.sh
```

服务会在后台运行,日志在 `nohup.out` 文件里。

## 📱 功能截图

*(建议添加以下截图)*
- 微信小程序首页
- 购票流程
- 车辆实时追踪
- 管理后台数据看板
- 订单管理界面

## 注意事项

1. `.env`、`cert/`、`密钥/` 这些敏感文件没提交到Git,部署时要自己配置
2. JWT密钥在生产环境一定要改掉,不要用默认的
3. SSL证书记得定期更新
4. 建议用HTTPS,特别是涉及支付的接口

## License

Copyright © 2024 狸猫日志. All rights reserved.

这是商业软件,别随便拿去商用哈。

## 有问题?

提Issue或者联系我都行:
- 微信: lihao68681818
- 邮箱: weizhiahao@163.com

## 感谢

用到的开源库:
- [Vue.js](https://vuejs.org/)
- [Element Plus](https://element-plus.org/)
- [Gin](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [Leaflet](https://leafletjs.com/)
- [ECharts](https://echarts.apache.org/)

---

觉得有用的话给个Star呗 ⭐
