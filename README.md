# 一个适用于VRChat的OSC小工具

具有如下功能

### 视奸

> 1. 电脑焦点窗口标题
> 2. 安卓设备前台应用
> 3. 电脑正在播放的媒体
> 4. 键盘长时间无输入显示AFK

### 其他

> 1. 电脑远程控制接口(远程关机)

### 特性

> 1. 自定义配置(data/config.json)：基本设置都在这里。需要重启OSC生效
> 2. 规则隐藏(data/rules_hide.json)：标题被匹配的项不会显示。需要重启OSC生效
> 3. 规则格式化(data/rules_title.json)：如果标题可被匹配则按所定格式显示，按顺序执行仅匹配第一次项。需要重启OSC生效
> 4. 规则替换(data/rules_hidetext.json)：标题中被匹配的项目会被替换为指定格式。需要重启OSC生效
> 5. 安卓应用展示名称(data/androidappkeys.json)：根据包名替换。即刻生效
> 6. 排除的安卓应用(data/androidappban.json)：根据包名。即刻生效

### data/config.json

```json
{
    // VRChat OSC 监听端口
    "vrchatoscport": 9000,

    // 全局信息刷新间隔（秒）
    "clock": 5,

    // 按下y后等待输入停止后重新显示（秒）
    "typingtimeout": 60,

    // 发送到 VRChat 时的前缀文本，如 "[💻]USING: Google Chrome"
    "applicationheadtext": "[💻]USING: ",

    // 键盘长时间无输入自动关机
    "autooffcomputer": false,
    // 超时时长（秒）
    "autooffcomputertimeout": 7200,

    // 手机应用检测配置
    "phone": {
        // 是否启用手机应用检测
        "status": true,
        // 手机应用的前缀文本，如 "[📱]USING: 微信"
        "applicationheadtext": "[📱]USING: "
        // 低电量展示
        "batteryheadtext": "BATTERY: ",
        // 低电量充电展示
        "batteryheadtextpower": "BATTERY: ⚡️"
    },

    // 音乐播放检测配置
    "music": {
        // 是否启用音乐检测
        "status": true,
        // 音乐信息的前缀文本，如 "MUSIC: 晴天 - 周杰伦"
        "headtext": "MUSIC: ",
        // 分割符
        "music_delimiter": "@"
        // 是否显示艺术家信息
        "showartist": true
    },

    // IoT 设备控制配置
    "iot": {
        // 是否启用 IoT 功能
        "status": true,
        // IoT 服务端口
        "port": 51851
    },

    // AFK（离开）自动检测配置
    "AFK": {
        // 是否启用 AFK 检测
        "status": true,
        // 多久无操作后触发 AFK（秒）
        "timeout": 900,
        // AFK 状态检测间隔（秒）
        "clock": 30
    }
}
```

### data/rules_hide.json

> 如果电脑前台应用标题被匹配则不会显示前台应用

```json
[
    {
        "name": "VSCode",
	// filename - project - Visual Studio Code
        "from": "^.*?\\s*?-\\s*?.*??\\s*?-\\s+(.*)$",
        "to": "[💻]USING: VSCode\n- PROJECT: $2\n- EDIT: $1",
        "rule": {
            "$3": "Visual Studio Code"
        }
    }
]
```

### data/rules_title.json

> 如果电脑前台应用标题被匹配则按照 to 替换为目标格式

```json
[
    {
        "name": "VSCode",
        "from": "^(.*?)\\s*?-\\s*?(.*?)?\\s*?-\\s+(.*)$",
        "to": "[💻]USING: VSCode\n- PROJECT: $2\n- EDIT: $1",
        "rule": {
            "$3": "Visual Studio Code"
        }
    },
    {
        "name": "Edge",
        "from": "^(.*?)(?: 和另外 \\d+ 个页面)? - 个人 - Microsoft Edge$",
        "to": "[💻]USING: Edge\n- VIEW: $1",
        "rule": {}
    },
    {
        "name": "VRCX-地雷版",
        "from": "^VRCX-Jirai \\d{4}.\\d{1,2}.\\d{1,2}$",
        "to": "[💻]USING: VRCX ( 地雷版 )",
        "rule": {}
    }
]
```

### data/rules_hidetext.json

> 被匹配的项目将被替换成 to

```json
[
    {
        "name": "IPv4",
        "from": "\\b(25[0-5]|2[0-4]\\d|[01]?\\d{1,2})\\.(25[0-5]|2[0-4]\\d|[01]?\\d{1,2})\\.(25[0-5]|2[0-4]\\d|[01]?\\d{1,2})\\.(25[0-5]|2[0-4]\\d|[01]?\\d{1,2})\\b",
        "to": "***.***.***.***"
    },
    {
        "name": "Host",
        "from": "(?i)^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z]{2,63}$",
        "to": "***.***"
    }
]
```

### data/androidappkays.json

> 根据包名替换手机应用展示的文本，支持概率随机

```json
{
    "com.android.camera": "相机",
    "mark.via": "Via",
    "com.tencent.mm": [
        { "content": "微信", "p": 0.7 },
        { "content": "小而美", "p": 0.2 },
        { "content": "绿色软件", "p": 0.1 }
    ],
    "com.tencent.mobileqq": [
        { "content": "QQ", "p": 0.2 },
        { "content": "QQ\n- 为什么没人找我聊天", "p": 0.2 },
        { "content": "QQ\n- 秦始皇: v我50...", "p": 0.2 },
        { "content": "企鹅", "p": 0.4 }
    ]
}
```

### data/androidappkays.jso

> 不显示的手机应用

```json
[
    "com.microsoft.appmanager"
]
```

### IOT API

关闭电脑（等待2分钟后自动关机，期间按任意键取消）

> http://127.0.0.1:51851/off_computer

取消关机 （如果当前无关机任务则跳过下次关机的指令）

> http://127.0.0.1:51851/stop_off_computer

### 规划

> 1. 支持以插件形式增加功能
> 2. 支持内容自定义排序
> 3. 支持全局正则

2026.5.18 - Made by Rexxrt.
