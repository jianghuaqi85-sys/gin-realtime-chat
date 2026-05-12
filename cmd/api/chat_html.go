package main

const chatHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>在线聊天室</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,"Helvetica Neue","Microsoft YaHei",sans-serif;height:100vh;overflow:hidden;background:linear-gradient(135deg,#0c0c1d 0%,#1a0a2e 30%,#2d1066 60%,#0c0c1d 100%);background-size:400% 400%;animation:gradBg 15s ease infinite}
@keyframes gradBg{0%{background-position:0% 50%}50%{background-position:100% 50%}100%{background-position:0% 50%}}

/* ===== 登录页 ===== */
.login-page{display:flex;align-items:center;justify-content:center;width:100%;height:100%;position:relative;overflow:hidden}
.login-page::before{content:'';position:absolute;width:500px;height:500px;background:radial-gradient(circle,rgba(244,114,182,.3),transparent 70%);top:-100px;right:-100px;animation:f1 8s ease-in-out infinite}
.login-page::after{content:'';position:absolute;width:400px;height:400px;background:radial-gradient(circle,rgba(129,140,248,.3),transparent 70%);bottom:-50px;left:-50px;animation:f2 10s ease-in-out infinite}
@keyframes f1{0%,100%{transform:translate(0,0)}50%{transform:translate(-30px,40px)}}
@keyframes f2{0%,100%{transform:translate(0,0)}50%{transform:translate(40px,-30px)}}
.login-box{background:rgba(255,255,255,.08);padding:48px 40px;border-radius:24px;box-shadow:0 20px 60px rgba(0,0,0,.4);width:380px;backdrop-filter:blur(30px);border:1px solid rgba(255,255,255,.15);position:relative;z-index:1}
.login-box .logo{width:64px;height:64px;background:linear-gradient(135deg,#f472b6,#818cf8);border-radius:16px;margin:0 auto 20px;display:flex;align-items:center;justify-content:center;font-size:28px;color:#fff;box-shadow:0 4px 20px rgba(244,114,182,.4)}
.login-box h2{margin-bottom:28px;text-align:center;color:#fff;font-size:22px;font-weight:600;text-shadow:0 2px 10px rgba(0,0,0,.3)}
.login-box input{width:100%;padding:14px 16px;margin-bottom:14px;border:1px solid rgba(255,255,255,.15);border-radius:14px;font-size:15px;outline:none;transition:all .3s;background:rgba(255,255,255,.08);color:#fff;backdrop-filter:blur(10px)}
.login-box input::placeholder{color:rgba(255,255,255,.4)}
.login-box input:focus{border-color:rgba(244,114,182,.6);box-shadow:0 0 0 3px rgba(244,114,182,.2),0 0 20px rgba(244,114,182,.1)}
.login-box button{width:100%;padding:14px;background:linear-gradient(135deg,#f472b6,#818cf8);color:#fff;border:none;border-radius:14px;font-size:15px;font-weight:500;cursor:pointer;transition:all .3s;box-shadow:0 4px 20px rgba(244,114,182,.3)}
.login-box button:hover{box-shadow:0 6px 25px rgba(244,114,182,.5);transform:translateY(-2px)}
.login-box button:disabled{background:rgba(255,255,255,.1);box-shadow:none;cursor:default;transform:none}
.login-box .switch{margin-top:16px;text-align:center;font-size:13px;color:rgba(255,255,255,.5)}
.login-box .switch a{color:#c4b5fd;cursor:pointer;font-weight:500}
.login-box .switch a:hover{color:#e9d5ff}
.login-box .error{color:#fca5a5;margin-top:10px;font-size:13px;text-align:center;min-height:20px}

/* ===== 主界面 ===== */
.app{display:none;width:100%;height:100%;position:relative}

/* 侧边栏 */
.sidebar{width:300px;background:rgba(255,255,255,.06);display:flex;flex-direction:column;border-right:1px solid rgba(255,255,255,.1);backdrop-filter:blur(30px)}
.sidebar-header{padding:20px;display:flex;align-items:center;gap:12px;border-bottom:1px solid rgba(255,255,255,.08)}
.sidebar-header .avatar{width:42px;height:42px;border-radius:50%;display:flex;align-items:center;justify-content:center;color:#fff;font-size:16px;font-weight:600;flex-shrink:0;box-shadow:0 2px 10px rgba(0,0,0,.2)}
.sidebar-header .user-meta{flex:1;min-width:0}
.sidebar-header .user-meta .name{color:#fff;font-size:15px;font-weight:500;text-shadow:0 1px 5px rgba(0,0,0,.2)}
.sidebar-header .user-meta .status{font-size:11px;color:#f472b6;margin-top:2px}
.channel-list{flex:1;overflow-y:auto;padding:8px}
.channel-list::-webkit-scrollbar{width:4px}
.channel-list::-webkit-scrollbar-thumb{background:rgba(255,255,255,.15);border-radius:2px}
.channel-list::-webkit-scrollbar-track{background:transparent}
.channel-item{display:flex;align-items:center;gap:12px;padding:12px 16px;border-radius:14px;cursor:pointer;transition:all .3s;margin-bottom:2px}
.channel-item:hover{background:rgba(255,255,255,.08)}
.channel-item.active{background:linear-gradient(135deg,rgba(244,114,182,.2),rgba(129,140,248,.2));box-shadow:0 0 20px rgba(244,114,182,.1);border:1px solid rgba(255,255,255,.1)}
.channel-item .ch-avatar{width:42px;height:42px;border-radius:12px;background:rgba(255,255,255,.08);display:flex;align-items:center;justify-content:center;font-size:18px;color:#c4b5fd;flex-shrink:0;transition:all .3s}
.channel-item.active .ch-avatar{background:linear-gradient(135deg,#f472b6,#818cf8);color:#fff;box-shadow:0 4px 15px rgba(244,114,182,.4)}
.channel-item .ch-info{flex:1;min-width:0}
.channel-item .ch-name{color:rgba(255,255,255,.8);font-size:14px;font-weight:500;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.channel-item.active .ch-name{color:#fff}
.channel-item .ch-preview{color:rgba(255,255,255,.4);font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;margin-top:2px}
.channel-item.active .ch-preview{color:rgba(255,255,255,.6)}
.new-channel{padding:12px;border-top:1px solid rgba(255,255,255,.08);display:flex;gap:8px}
.new-channel input{flex:1;padding:10px 12px;background:rgba(255,255,255,.08);border:1px solid rgba(255,255,255,.1);border-radius:12px;font-size:13px;color:#fff;outline:none;transition:all .3s;backdrop-filter:blur(10px)}
.new-channel input::placeholder{color:rgba(255,255,255,.3)}
.new-channel input:focus{border-color:rgba(244,114,182,.5);box-shadow:0 0 0 3px rgba(244,114,182,.15)}
.new-channel button{width:40px;background:linear-gradient(135deg,#f472b6,#818cf8);color:#fff;border:none;border-radius:12px;font-size:18px;cursor:pointer;transition:all .3s;box-shadow:0 2px 10px rgba(244,114,182,.3)}
.new-channel button:hover{box-shadow:0 4px 15px rgba(244,114,182,.5);transform:scale(1.05)}

/* 聊天区 */
.chat-area{flex:1;display:flex;flex-direction:column;min-width:0}
.chat-header{padding:16px 24px;background:rgba(255,255,255,.04);border-bottom:1px solid rgba(255,255,255,.08);font-weight:600;font-size:16px;color:#fff;display:flex;align-items:center;gap:8px;backdrop-filter:blur(20px)}
.chat-header .dot{width:8px;height:8px;border-radius:50%;flex-shrink:0}
.chat-header .dot.on{background:#f472b6;box-shadow:0 0 10px rgba(244,114,182,.6)}
.chat-header .dot.off{background:#f87171;box-shadow:0 0 8px rgba(248,113,113,.4)}
.chat-header .ch-title{flex:1}
.chat-header .ch-status{font-size:11px;color:rgba(255,255,255,.4);font-weight:400}

/* 消息区域 */
.messages{flex:1;overflow-y:auto;padding:20px 24px}
.messages::-webkit-scrollbar{width:6px}
.messages::-webkit-scrollbar-thumb{background:rgba(255,255,255,.1);border-radius:3px}
.messages::-webkit-scrollbar-track{background:transparent}

/* 时间戳 */
.msg-time{text-align:center;margin:16px 0}
.msg-time span{background:rgba(255,255,255,.08);color:rgba(255,255,255,.5);font-size:11px;padding:4px 14px;border-radius:20px;backdrop-filter:blur(10px)}

/* 系统消息 */
.msg-system{text-align:center;margin:10px 0}
.msg-system span{color:rgba(255,255,255,.4);font-size:12px}

/* 对方消息 */
.msg-row{display:flex;gap:10px;margin-bottom:16px;align-items:flex-start}
.msg-row .avatar{width:38px;height:38px;border-radius:50%;display:flex;align-items:center;justify-content:center;color:#fff;font-size:13px;font-weight:600;flex-shrink:0;margin-top:2px;box-shadow:0 2px 10px rgba(0,0,0,.2)}
.msg-row .content{max-width:60%}
.msg-row .content .name{font-size:11px;color:rgba(255,255,255,.4);margin-bottom:4px;padding-left:2px}
.msg-row .bubble{background:rgba(255,255,255,.1);padding:12px 16px;border-radius:18px;font-size:14px;line-height:1.6;word-break:break-word;position:relative;color:rgba(255,255,255,.9);backdrop-filter:blur(20px);border:1px solid rgba(255,255,255,.1)}
.msg-row .bubble::before{content:'';position:absolute;left:-7px;top:14px;border:7px solid transparent;border-right-color:rgba(255,255,255,.1)}

/* 自己消息 */
.msg-row.self{flex-direction:row-reverse}
.msg-row.self .content{text-align:right}
.msg-row.self .content .name{display:none}
.msg-row.self .bubble{background:linear-gradient(135deg,rgba(244,114,182,.4),rgba(129,140,248,.4));color:#fff;border:1px solid rgba(255,255,255,.2);box-shadow:0 4px 20px rgba(244,114,182,.2);backdrop-filter:blur(20px)}
.msg-row.self .bubble::before{left:auto;right:-7px;border-right-color:transparent;border-left-color:rgba(244,114,182,.4)}

/* 输入区 */
.input-area{padding:16px 24px;background:rgba(255,255,255,.04);border-top:1px solid rgba(255,255,255,.08);display:flex;gap:12px;align-items:flex-end;backdrop-filter:blur(20px)}
.input-area input{flex:1;padding:12px 16px;border:1px solid rgba(255,255,255,.12);border-radius:14px;font-size:14px;outline:none;transition:all .3s;background:rgba(255,255,255,.08);color:#fff;backdrop-filter:blur(10px)}
.input-area input:focus{border-color:rgba(244,114,182,.5);box-shadow:0 0 0 3px rgba(244,114,182,.15),0 0 20px rgba(244,114,182,.05)}
.input-area input::placeholder{color:rgba(255,255,255,.3)}
.input-area button{padding:12px 28px;background:linear-gradient(135deg,#f472b6,#818cf8);color:#fff;border:none;border-radius:14px;font-size:14px;font-weight:500;cursor:pointer;transition:all .3s;white-space:nowrap;box-shadow:0 4px 15px rgba(244,114,182,.3)}
.input-area button:hover{box-shadow:0 6px 20px rgba(244,114,182,.5);transform:translateY(-1px)}
.input-area button:disabled{background:rgba(255,255,255,.1);box-shadow:none;cursor:default;transform:none}

/* 管理面板 */
.admin-btn{padding:12px 16px;border-top:1px solid rgba(255,255,255,.06);display:flex;align-items:center;gap:10px;cursor:pointer;color:rgba(255,255,255,.5);font-size:13px;transition:all .2s}
.admin-btn:hover{background:rgba(255,255,255,.06);color:#c4b5fd}
.admin-btn .icon{font-size:16px}
.admin-panel{display:none;position:absolute;top:0;left:300px;right:0;bottom:0;z-index:20;overflow-y:auto;backdrop-filter:blur(30px)}
.admin-panel.show{display:block}
.admin-panel .panel-header{padding:20px 24px;background:rgba(255,255,255,.04);border-bottom:1px solid rgba(255,255,255,.08);display:flex;align-items:center;justify-content:space-between;backdrop-filter:blur(20px)}
.admin-panel .panel-header h3{font-size:18px;color:#fff}
.admin-panel .panel-header .close{width:32px;height:32px;border-radius:50%;border:none;background:rgba(255,255,255,.1);cursor:pointer;font-size:18px;display:flex;align-items:center;justify-content:center;color:#fff;transition:all .2s;backdrop-filter:blur(10px)}
.admin-panel .panel-header .close:hover{background:rgba(244,114,182,.2)}
.admin-tabs{display:flex;border-bottom:1px solid rgba(255,255,255,.08);background:rgba(255,255,255,.02);padding:0 24px}
.admin-tabs .tab{padding:12px 20px;cursor:pointer;font-size:13px;color:rgba(255,255,255,.4);border-bottom:2px solid transparent;transition:all .2s}
.admin-tabs .tab:hover{color:rgba(255,255,255,.7)}
.admin-tabs .tab.active{color:#f472b6;border-bottom-color:#f472b6}
.admin-content{padding:24px}
.admin-content .section{display:none}
.admin-content .section.show{display:block}

/* 统计卡片 */
.stats-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:24px}
.stat-card{background:rgba(255,255,255,.08);border-radius:16px;padding:20px;text-align:center;border:1px solid rgba(255,255,255,.1);transition:all .3s;backdrop-filter:blur(15px)}
.stat-card:hover{background:rgba(255,255,255,.12);transform:translateY(-3px);box-shadow:0 8px 25px rgba(0,0,0,.2)}
.stat-card .num{font-size:28px;font-weight:700;background:linear-gradient(135deg,#f472b6,#818cf8);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
.stat-card .label{font-size:12px;color:rgba(255,255,255,.4);margin-top:4px}

/* 用户列表 */
.user-list{display:flex;flex-direction:column;gap:8px}
.user-row{display:flex;align-items:center;gap:12px;padding:12px 16px;background:rgba(255,255,255,.06);border-radius:14px;border:1px solid rgba(255,255,255,.08);transition:all .2s;backdrop-filter:blur(10px)}
.user-row:hover{background:rgba(255,255,255,.1)}
.user-row .info{flex:1}
.user-row .info .name{font-size:14px;font-weight:500;color:#fff}
.user-row .info .role{font-size:11px;color:rgba(255,255,255,.4)}
.user-row .info .banned{font-size:11px;color:#fca5a5}
.user-row .actions{display:flex;gap:6px}
.user-row .actions button{padding:6px 12px;border:none;border-radius:8px;font-size:12px;cursor:pointer;transition:all .2s;backdrop-filter:blur(10px)}
.btn-del-user{background:rgba(248,113,113,.2);color:#fca5a5}.btn-del-user:hover{background:rgba(248,113,113,.3)}
.btn-ban{background:rgba(248,113,113,.2);color:#fca5a5}.btn-ban:hover{background:rgba(248,113,113,.3)}
.btn-unban{background:rgba(167,139,250,.2);color:#c4b5fd}.btn-unban:hover{background:rgba(167,139,250,.3)}
.btn-del{background:rgba(248,113,113,.2);color:#fca5a5}.btn-del:hover{background:rgba(248,113,113,.3)}
.btn-clear{background:rgba(251,146,60,.2);color:#fdba74}.btn-clear:hover{background:rgba(251,146,60,.3)}

/* 频道管理 */
.ch-list{display:flex;flex-direction:column;gap:8px}
.ch-row{display:flex;align-items:center;gap:12px;padding:12px 16px;background:rgba(255,255,255,.06);border-radius:14px;border:1px solid rgba(255,255,255,.08)}
.ch-row .info{flex:1}
.ch-row .info .name{font-size:14px;font-weight:500;color:#fff}
.ch-row .info .meta{font-size:11px;color:rgba(255,255,255,.4)}

/* 广播 */
.broadcast-area{display:flex;gap:12px}
.broadcast-area input{flex:1;padding:12px;border:1px solid rgba(255,255,255,.12);border-radius:12px;font-size:14px;background:rgba(255,255,255,.08);color:#fff;outline:none;backdrop-filter:blur(10px)}
.broadcast-area input:focus{border-color:rgba(244,114,182,.5)}
.broadcast-area button{padding:12px 24px;background:linear-gradient(135deg,#ef4444,#dc2626);color:#fff;border:none;border-radius:12px;cursor:pointer;transition:all .3s;box-shadow:0 4px 15px rgba(239,68,68,.3)}
.broadcast-area button:hover{box-shadow:0 6px 20px rgba(239,68,68,.4)}

/* 分享地址 */
.tunnel-row{display:flex;align-items:center;gap:6px;margin-top:4px}
.tunnel-row .url{flex:1;font-size:11px;color:rgba(255,255,255,.4);word-break:break-all;font-family:monospace}
.tunnel-row .copy-btn{padding:2px 10px;background:rgba(255,255,255,.1);color:#c4b5fd;border:none;border-radius:8px;font-size:10px;cursor:pointer;white-space:nowrap;transition:all .2s;backdrop-filter:blur(10px)}
.tunnel-row .copy-btn:hover{background:rgba(255,255,255,.15)}
.tunnel-row .copy-btn.copied{background:rgba(255,255,255,.05);color:rgba(255,255,255,.3)}

/* 在线用户 */
.online-users{padding:8px 16px;border-top:1px solid rgba(255,255,255,.06)}
.online-title{font-size:11px;color:rgba(255,255,255,.3);margin-bottom:6px}
.online-list{display:flex;flex-wrap:wrap;gap:4px}
.online-tag{display:inline-flex;align-items:center;gap:4px;padding:2px 8px;background:rgba(255,255,255,.08);border-radius:10px;font-size:11px;color:rgba(255,255,255,.6);backdrop-filter:blur(10px)}
.online-tag .dot{width:6px;height:6px;border-radius:50%;background:#f472b6;box-shadow:0 0 8px rgba(244,114,182,.5)}

/* 修改密码弹窗 */
.modal-overlay{display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.5);z-index:100;align-items:center;justify-content:center;backdrop-filter:blur(8px)}
.modal-overlay.show{display:flex}
.modal-box{background:rgba(255,255,255,.1);border-radius:24px;padding:28px;width:360px;max-width:90%;border:1px solid rgba(255,255,255,.15);backdrop-filter:blur(30px);box-shadow:0 20px 60px rgba(0,0,0,.3)}
.modal-box h3{margin-bottom:20px;font-size:16px;color:#fff}
.modal-box input{width:100%;padding:12px;border:1px solid rgba(255,255,255,.15);border-radius:12px;font-size:14px;margin-bottom:12px;outline:none;background:rgba(255,255,255,.08);color:#fff;backdrop-filter:blur(10px);transition:all .3s}
.modal-box input::placeholder{color:rgba(255,255,255,.3)}
.modal-box input:focus{border-color:rgba(244,114,182,.5);box-shadow:0 0 0 3px rgba(244,114,182,.15)}
.modal-box .modal-btns{display:flex;gap:10px;margin-top:8px}
.modal-box .modal-btns button{flex:1;padding:10px;border:none;border-radius:12px;font-size:14px;cursor:pointer;transition:all .3s}
.modal-box .btn-cancel{background:rgba(255,255,255,.1);color:rgba(255,255,255,.6);backdrop-filter:blur(10px)}
.modal-box .btn-cancel:hover{background:rgba(255,255,255,.15)}
.modal-box .btn-confirm{background:linear-gradient(135deg,#f472b6,#818cf8);color:#fff;box-shadow:0 4px 15px rgba(244,114,182,.3)}
.modal-box .btn-confirm:hover{box-shadow:0 6px 20px rgba(244,114,182,.5)}

/* 消息操作菜单 */
.msg-actions{display:none;position:absolute;top:-30px;right:0;background:rgba(255,255,255,.12);border-radius:12px;box-shadow:0 4px 20px rgba(0,0,0,.3);overflow:hidden;z-index:10;border:1px solid rgba(255,255,255,.15);backdrop-filter:blur(20px)}
.msg-row.self .msg-actions{left:0;right:auto}
.msg-actions button{display:block;padding:6px 14px;border:none;background:none;font-size:12px;cursor:pointer;white-space:nowrap;color:rgba(255,255,255,.8);transition:all .15s}
.msg-actions button:hover{background:rgba(255,255,255,.1)}
.msg-actions button.danger{color:#fca5a5}
.msg-actions button.danger:hover{background:rgba(248,113,113,.15)}

/* 加载更多 */
.load-more{text-align:center;padding:12px;color:rgba(255,255,255,.4);font-size:12px;cursor:pointer;transition:color .2s}
.load-more:hover{color:#c4b5fd}
.loading-spinner{display:inline-block;width:16px;height:16px;border:2px solid rgba(255,255,255,.15);border-top-color:#f472b6;border-radius:50%;animation:spin .6s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}

/* 汉堡菜单按钮 */
.menu-btn{display:none;width:32px;height:32px;border:none;background:none;font-size:20px;cursor:pointer;color:#fff;flex-shrink:0;padding:0;line-height:1}

/* 响应式 */
@media(max-width:768px){
  .sidebar{width:100%;position:absolute;z-index:10;height:100%;left:0;top:0}
  .sidebar.hidden{display:none}
  .chat-area{width:100%}
  .msg-row .content{max-width:75%}
  .menu-btn{display:block}
  .admin-panel{left:0}
}
</style>
</head>
<body>

<!-- 登录页 -->
<div class="login-page" id="loginPage">
<div class="login-box">
  <div class="logo">💬</div>
  <h2 id="formTitle">登录</h2>
  <input id="authUser" placeholder="用户名" autocomplete="username">
  <input id="authPass" placeholder="密码" type="password" autocomplete="current-password">
  <input id="authPass2" placeholder="确认密码" type="password" style="display:none">
  <button onclick="doAuth()" id="authBtn">登录</button>
  <div class="switch">
    <span id="switchText">没有账号？</span> <a onclick="toggleMode()" id="switchLink">注册</a>
  </div>
  <div class="error" id="authError"></div>
</div>
</div>

<!-- 修改密码弹窗 -->
<div class="modal-overlay" id="pwdModal">
  <div class="modal-box">
    <h3>修改密码</h3>
    <input id="oldPwd" type="password" placeholder="当前密码">
    <input id="newPwd" type="password" placeholder="新密码（至少 8 位）">
    <input id="newPwd2" type="password" placeholder="确认新密码">
    <div class="error" id="pwdError" style="min-height:18px;font-size:13px;margin-bottom:8px"></div>
    <div class="modal-btns">
      <button class="btn-cancel" onclick="closeChangePassword()">取消</button>
      <button class="btn-confirm" onclick="doChangePassword()">确认修改</button>
    </div>
  </div>
</div>

<!-- 主界面 -->
<div class="app" id="app">
<div class="sidebar" id="sidebar">
  <div class="sidebar-header">
    <div class="avatar" id="myAvatar"></div>
    <div class="user-meta">
      <div class="name" id="myName"></div>
      <div class="status">在线</div>
      <div id="tunnelArea" style="display:none;margin-top:6px">
        <div id="tunnelUrls"></div>
      </div>
    </div>
  </div>
  <div class="channel-list" id="channelList"></div>
  <div class="new-channel">
    <input id="newChannelName" placeholder="新建频道..." onkeydown="if(event.key==='Enter')createChannel()">
    <button onclick="createChannel()">+</button>
  </div>
  <div class="online-users" id="onlineUsers">
    <div class="online-title">在线用户 (<span id="onlineCount">0</span>)</div>
    <div class="online-list" id="onlineList"></div>
  </div>
  <div class="admin-btn" onclick="showChangePassword()">
    <span class="icon">🔑</span> 修改密码
  </div>
  <div class="admin-btn" id="adminBtn" onclick="toggleAdmin()" style="display:none">
    <span class="icon">⚙</span> 管理面板
  </div>
</div>
<div class="chat-area">
  <div class="chat-header">
    <button class="menu-btn" onclick="toggleSidebar()">&#9776;</button>
    <span class="dot" id="wsDot"></span>
    <span class="ch-title" id="chatTitle">选择一个频道</span>
    <span class="ch-status" id="chatStatus"></span>
  </div>
  <div class="messages" id="messages">
    <div style="display:flex;align-items:center;justify-content:center;height:100%;color:#bbb">
      <div style="text-align:center"><div style="font-size:48px;margin-bottom:12px">💬</div><div>选择一个频道开始聊天</div></div>
    </div>
  </div>
  <div class="input-area">
    <input id="msgInput" placeholder="输入消息..." onkeydown="if(event.key==='Enter')sendMessage()" disabled>
    <button onclick="sendMessage()" id="sendBtn" disabled>发送</button>
  </div>
</div>

<!-- 管理面板 -->
<div class="admin-panel" id="adminPanel">
  <div class="panel-header">
    <h3>管理面板</h3>
    <button class="close" onclick="toggleAdmin()">&times;</button>
  </div>
  <div class="admin-tabs">
    <div class="tab active" onclick="switchAdminTab('stats',this)">服务器状态</div>
    <div class="tab" onclick="switchAdminTab('users',this)">用户管理</div>
    <div class="tab" onclick="switchAdminTab('channels',this)">频道管理</div>
    <div class="tab" onclick="switchAdminTab('broadcast',this)">广播公告</div>
  </div>
  <div class="admin-content">
    <div class="section show" id="sec-stats">
      <div class="stats-grid" id="statsGrid"></div>
    </div>
    <div class="section" id="sec-users">
      <div class="user-list" id="userList"></div>
    </div>
    <div class="section" id="sec-channels">
      <div class="ch-list" id="adminChannelList"></div>
    </div>
    <div class="section" id="sec-broadcast">
      <div class="broadcast-area">
        <input id="broadcastInput" placeholder="输入公告内容...">
        <button onclick="adminBroadcast()">广播给所有人</button>
      </div>
    </div>
  </div>
</div>
</div>

<script>
let token='',ws=null,currentChannel='',currentChannelName='',isRegister=false,myUsername='',lastMsgTime=0;
let wsReconnectDelay=1000,wsReconnectAttempts=0,wsConnected=false,pollTimer=null;
const API=location.origin;

// 头像颜色池
const AVATAR_COLORS=['#07C160','#FA5151','#1989FA','#FF8800','#6467EF','#00B578','#FF6770','#576B95'];
function avatarColor(name){let h=0;for(let i=0;i<name.length;i++)h=name.charCodeAt(i)+((h<<5)-h);return AVATAR_COLORS[Math.abs(h)%AVATAR_COLORS.length];}
function avatarHTML(name){const d=document.createElement('div');d.textContent=name.charAt(0).toUpperCase();return '<div class="avatar" style="background:'+avatarColor(name)+'">'+d.innerHTML+'</div>';}

// 时间格式化
function formatTime(ts){
  if(!ts)return '';
  const d=new Date(ts);
  const now=new Date();
  const pad=n=>String(n).padStart(2,'0');
  if(d.toDateString()===now.toDateString())return pad(d.getHours())+':'+pad(d.getMinutes());
  const yesterday=new Date(now);yesterday.setDate(yesterday.getDate()-1);
  if(d.toDateString()===yesterday.toDateString())return '昨天 '+pad(d.getHours())+':'+pad(d.getMinutes());
  return (d.getMonth()+1)+'/'+d.getDate()+' '+pad(d.getHours())+':'+pad(d.getMinutes());
}
function shouldShowTime(ts){
  if(!ts)return true;
  const d=new Date(ts).getTime();
  if(d-lastMsgTime>5*60*1000){lastMsgTime=d;return true;}
  return false;
}

// 登录/注册
function toggleMode(){
  isRegister=!isRegister;
  document.getElementById('formTitle').textContent=isRegister?'注册':'登录';
  document.getElementById('authBtn').textContent=isRegister?'注册':'登录';
  document.getElementById('switchText').textContent=isRegister?'已有账号？':'没有账号？';
  document.getElementById('switchLink').textContent=isRegister?'登录':'注册';
  document.getElementById('authPass2').style.display=isRegister?'block':'none';
  document.getElementById('authError').textContent='';
}
async function doAuth(){
  const u=document.getElementById('authUser').value;
  const p=document.getElementById('authPass').value;
  const btn=document.getElementById('authBtn');
  document.getElementById('authError').textContent='';
  btn.disabled=true;
  try{
    if(isRegister){
      const p2=document.getElementById('authPass2').value;
      if(p!==p2){document.getElementById('authError').style.color='#ef4444';document.getElementById('authError').textContent='两次输入的密码不一致';return;}
      btn.textContent='注册中...';
      const r=await fetch(API+'/api/public/register',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:p,confirm_password:p2})});
      const d=await r.json();
      if(r.ok){toggleMode();document.getElementById('authError').style.color='#07C160';document.getElementById('authError').textContent='注册成功，请登录';}
      else{document.getElementById('authError').style.color='#ef4444';document.getElementById('authError').textContent=d.error||'注册失败';}
      return;
    }
    btn.textContent='登录中...';
    const r=await fetch(API+'/api/public/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username:u,password:p})});
    const d=await r.json();
    if(d.token){token=d.token;myUsername=u;localStorage.setItem('token',token);localStorage.setItem('username',myUsername);startApp();}
    else{document.getElementById('authError').textContent=d.error||'登录失败';}
  }finally{
    btn.disabled=false;
    btn.textContent=isRegister?'注册':'登录';
  }
}

async function startApp(){
  document.getElementById('loginPage').style.display='none';
  document.getElementById('app').style.display='flex';
  document.getElementById('myAvatar').outerHTML=avatarHTML(myUsername).replace('class="avatar"','class="avatar" id="myAvatar"');
  document.getElementById('myName').textContent=myUsername;
  await loadChannels();
  connectWS();
  checkAdmin();
  loadTunnel();
  loadOnlineUsers();
  setupMessageScroll();
  // 每 30 秒刷新在线用户
  setInterval(loadOnlineUsers,30000);
}
async function checkAdmin(){
  try{
    const r=await fetch(API+'/api/admin/stats',{headers:{'Authorization':'Bearer '+token}});
    if(r.ok)document.getElementById('adminBtn').style.display='flex';
  }catch(e){}
}

// 频道
async function loadChannels(){
  const r=await fetch(API+'/api/channels',{headers:{'Authorization':'Bearer '+token}});
  const channels=await r.json();
  const el=document.getElementById('channelList');
  el.innerHTML='';
  channels.forEach(ch=>{
    const div=document.createElement('div');
    div.className='channel-item'+(ch.ID===currentChannel?' active':'');
    div.innerHTML='<div class="ch-avatar">#</div><div class="ch-info"><div class="ch-name">'+esc(ch.Name)+'</div><div class="ch-preview">频道</div></div>';
    div.onclick=()=>selectChannel(ch.ID,ch.Name);
    el.appendChild(div);
  });
}
async function createChannel(){
  const input=document.getElementById('newChannelName');
  const name=input.value.trim();
  if(!name)return;
  input.disabled=true;
  try{
    await fetch(API+'/api/channels',{method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify({name})});
    input.value='';
    await loadChannels();
  }finally{
    input.disabled=false;
    input.focus();
  }
}

function toggleSidebar(){
  document.getElementById('sidebar').classList.toggle('hidden');
}
function selectChannel(id,name){
  if(currentChannel===id)return;
  if(currentChannel&&ws&&ws.readyState===1)ws.send(JSON.stringify({type:'leave',channel_id:currentChannel}));
  currentChannel=id;currentChannelName=name;
  document.getElementById('chatTitle').textContent=name;
  document.getElementById('msgInput').disabled=false;
  document.getElementById('sendBtn').disabled=false;
  document.querySelectorAll('.channel-item').forEach(el=>el.classList.remove('active'));
  if(event&&event.target){const t=event.target.closest('.channel-item');if(t)t.classList.add('active');}
  document.getElementById('messages').innerHTML='';
  lastMsgTime=0;firstMsgTime='';hasMoreMessages=true;
  // 移动端选择频道后自动隐藏侧边栏
  if(window.innerWidth<=768)document.getElementById('sidebar').classList.add('hidden');
  loadHistory(id);
  if(ws&&ws.readyState===1)ws.send(JSON.stringify({type:'join',channel_id:id}));
}

async function loadHistory(channelId){
  const r=await fetch(API+'/api/channels/'+channelId+'/messages?limit=100',{headers:{'Authorization':'Bearer '+token}});
  const msgs=await r.json();
  const el=document.getElementById('messages');el.innerHTML='';
  lastMsgTime=0;
  if(msgs.length>0)firstMsgTime=msgs[0].CreatedAt;
  hasMoreMessages=msgs.length>=100;
  msgs.forEach(m=>{
    if(shouldShowTime(m.CreatedAt))appendTime(formatTime(m.CreatedAt));
    appendMsg(m.UserID,m.Username,m.Content,m.ID);
  });
  el.scrollTop=el.scrollHeight;
}

// WebSocket — 指数退避重连 + 降级轮询
function connectWS(){
  if(ws&&(ws.readyState===0||ws.readyState===1))return;
  const proto=location.protocol==='https:'?'wss':'ws';
  ws=new WebSocket(proto+'://'+location.host+'/api/ws');
  ws.onopen=()=>{
    ws.send(JSON.stringify({type:'auth',token:token}));
  };
  ws.onclose=()=>{
    wsConnected=false;
    document.getElementById('wsDot').className='dot off';
    if(wsReconnectAttempts===0){
      document.getElementById('chatStatus').textContent='连接断开，正在重连...';
    }
    // 指数退避：1s → 2s → 4s → 8s → 16s → 30s
    const delay=Math.min(wsReconnectDelay*Math.pow(2,wsReconnectAttempts),30000);
    wsReconnectAttempts++;
    // 超过 5 次重连失败，启动降级轮询
    if(wsReconnectAttempts>5&&!pollTimer){
      document.getElementById('chatStatus').textContent='连接失败，使用轮询模式';
      startPolling();
    }
    setTimeout(connectWS,delay);
  };
  ws.onmessage=(e)=>{
    let msg;
    try{msg=JSON.parse(e.data);}catch(err){return;}
    if(msg.type==='auth_ok'){
      wsConnected=true;
      wsReconnectAttempts=0;
      wsReconnectDelay=1000;
      document.getElementById('wsDot').className='dot on';
      document.getElementById('chatStatus').textContent='已连接';
      stopPolling();
      if(currentChannel)ws.send(JSON.stringify({type:'join',channel_id:currentChannel}));
    }else if(msg.type==='error'){
      document.getElementById('wsDot').className='dot off';
      document.getElementById('chatStatus').textContent=msg.content||'连接错误';
      ws.close();
    }else if(msg.type==='message'&&msg.channel_id===currentChannel){
      const el=document.getElementById('messages');
      if(shouldShowTime(msg.created_at))appendTime(formatTime(msg.created_at));
      appendMsg(msg.user_id,msg.username,msg.content);
      el.scrollTop=el.scrollHeight;
    }else if(msg.type==='system'){
      if(!msg.channel_id||msg.channel_id===currentChannel){
        appendSystem(msg.content);
        document.getElementById('messages').scrollTop=document.getElementById('messages').scrollHeight;
      }
    }
  };
}

// 降级轮询：WS 连不上时定期拉取新消息
function startPolling(){
  if(pollTimer)return;
  pollTimer=setInterval(async()=>{
    if(!currentChannel||!token)return;
    try{
      const r=await fetch(API+'/api/channels/'+currentChannel+'/messages?limit=20',{headers:{'Authorization':'Bearer '+token}});
      if(!r.ok)return;
      const msgs=await r.json();
      const el=document.getElementById('messages');
      const existingCount=el.querySelectorAll('.msg-row').length;
      if(msgs.length>existingCount){
        el.innerHTML='';
        lastMsgTime=0;
        msgs.forEach(m=>{
          if(shouldShowTime(m.CreatedAt))appendTime(formatTime(m.CreatedAt));
          appendMsg(m.UserID,m.Username,m.Content);
        });
        el.scrollTop=el.scrollHeight;
      }
    }catch(e){}
  },5000);
}
function stopPolling(){
  if(pollTimer){clearInterval(pollTimer);pollTimer=null;}
}

// 消息渲染
function appendTime(text){
  const el=document.getElementById('messages');
  const div=document.createElement('div');
  div.className='msg-time';
  div.innerHTML='<span>'+text+'</span>';
  el.appendChild(div);
}
function appendMsg(userId,username,content,msgId){
  document.getElementById('messages').appendChild(createMsgElement(userId,username,content,msgId));
}
function appendSystem(content){
  const el=document.getElementById('messages');
  const div=document.createElement('div');
  div.className='msg-system';
  div.innerHTML='<span>'+esc(content)+'</span>';
  el.appendChild(div);
}

function sendMessage(){
  const input=document.getElementById('msgInput');
  const content=input.value.trim();
  if(!content||!currentChannel)return;
  if(!ws||ws.readyState!==1||!wsConnected){
    appendSystem('消息发送失败：连接已断开');
    document.getElementById('messages').scrollTop=document.getElementById('messages').scrollHeight;
    return;
  }
  ws.send(JSON.stringify({type:'message',channel_id:currentChannel,content}));
  input.value='';
  input.focus();
}

function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML;}

// ===== 管理面板 =====
function toggleAdmin(){
  const p=document.getElementById('adminPanel');
  p.classList.toggle('show');
  if(p.classList.contains('show'))loadAdminStats();
}
function switchAdminTab(name,el){
  document.querySelectorAll('.admin-tabs .tab').forEach(t=>t.classList.remove('active'));
  el.classList.add('active');
  document.querySelectorAll('.admin-content .section').forEach(s=>s.classList.remove('show'));
  document.getElementById('sec-'+name).classList.add('show');
  if(name==='stats')loadAdminStats();
  if(name==='users')loadAdminUsers();
  if(name==='channels')loadAdminChannels();
}

async function loadAdminStats(){
  const r=await fetch(API+'/api/admin/stats',{headers:{'Authorization':'Bearer '+token}});
  const d=await r.json();
  document.getElementById('statsGrid').innerHTML=
    '<div class="stat-card"><div class="num">'+d.online+'</div><div class="label">在线人数</div></div>'+
    '<div class="stat-card"><div class="num">'+d.users_total+'</div><div class="label">注册用户</div></div>'+
    '<div class="stat-card"><div class="num">'+d.channels+'</div><div class="label">频道数</div></div>'+
    '<div class="stat-card"><div class="num">'+d.messages+'</div><div class="label">消息总数</div></div>';
}
async function loadTunnel(){
  const area=document.getElementById('tunnelArea');
  const container=document.getElementById('tunnelUrls');
  try{
    const r=await fetch(API+'/api/tunnel',{headers:{'Authorization':'Bearer '+token}});
    if(!r.ok){area.style.display='none';return;}
    const d=await r.json();
    if(!d.urls||d.urls.length===0){area.style.display='none';return;}
    container.innerHTML='';
    d.urls.forEach(u=>{
      const row=document.createElement('div');
      row.className='tunnel-row';
      row.innerHTML='<span class="url">'+esc(u)+'</span><button class="copy-btn" onclick="copyTunnelUrl(this,\''+esc(u)+'\')">复制</button>';
      container.appendChild(row);
    });
    area.style.display='block';
  }catch(e){area.style.display='none';}
}
function copyTunnelUrl(btn,url){
  navigator.clipboard.writeText(url).then(()=>{
    btn.textContent='已复制';
    btn.classList.add('copied');
    setTimeout(()=>{btn.textContent='复制';btn.classList.remove('copied');},2000);
  });
}

async function loadAdminUsers(){
  const r=await fetch(API+'/api/admin/users',{headers:{'Authorization':'Bearer '+token}});
  const users=await r.json();
  const el=document.getElementById('userList');el.innerHTML='';
  users.forEach(u=>{
    const div=document.createElement('div');
    div.className='user-row';
    const roleLabel=u.role==='admin'?'<span style="color:#07C160">管理员</span>':'普通用户';
    const banLabel=u.banned?'<span class="banned">已封禁</span>':'';
    const actions=u.username!=='admin'?(
      '<button class="btn-del-user" onclick="adminDeleteUser(\''+u.id+'\',\''+esc(u.username)+'\')">删除</button>'+
      (u.banned?'<button class="btn-unban" onclick="adminUnban(\''+u.id+'\')">解封</button>':'<button class="btn-ban" onclick="adminBan(\''+u.id+'\')">封禁</button>')
    ):'';
    div.innerHTML=avatarHTML(u.username)+'<div class="info"><div class="name">'+esc(u.username)+'</div><div class="role">'+roleLabel+' '+banLabel+'</div></div><div class="actions">'+actions+'</div>';
    el.appendChild(div);
  });
}

async function loadAdminChannels(){
  const r=await fetch(API+'/api/channels',{headers:{'Authorization':'Bearer '+token}});
  const channels=await r.json();
  const el=document.getElementById('adminChannelList');el.innerHTML='';
  channels.forEach(ch=>{
    const div=document.createElement('div');
    div.className='ch-row';
    div.innerHTML='<div class="info"><div class="name"># '+esc(ch.Name)+'</div><div class="meta">ID: '+ch.ID+'</div></div>'+
      '<div class="actions">'+
      '<button class="btn-clear" onclick="adminClearMsg(\''+ch.ID+'\')">清空消息</button>'+
      '<button class="btn-del" onclick="adminDelChannel(\''+ch.ID+'\')">删除频道</button>'+
      '</div>';
    el.appendChild(div);
  });
}

async function adminDeleteUser(uid,name){
  if(!confirm('确定永久删除用户「'+name+'」？该操作不可恢复！'))return;
  await fetch(API+'/api/admin/users/'+uid,{method:'DELETE',headers:{'Authorization':'Bearer '+token}});
  loadAdminUsers();loadAdminStats();
}
async function adminBan(uid){
  if(!confirm('确定封禁该用户？'))return;
  await fetch(API+'/api/admin/ban',{method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify({user_id:uid})});
  loadAdminUsers();
}
async function adminUnban(uid){
  await fetch(API+'/api/admin/unban',{method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify({user_id:uid})});
  loadAdminUsers();
}
async function adminDelChannel(id){
  if(!confirm('确定删除该频道及所有消息？'))return;
  await fetch(API+'/api/admin/channels/'+id,{method:'DELETE',headers:{'Authorization':'Bearer '+token}});
  loadAdminChannels();loadAdminStats();loadChannels();
}
async function adminClearMsg(id){
  if(!confirm('确定清空该频道所有消息？'))return;
  await fetch(API+'/api/admin/channels/'+id+'/messages',{method:'DELETE',headers:{'Authorization':'Bearer '+token}});
  if(currentChannel===id){document.getElementById('messages').innerHTML='';}
}
async function adminBroadcast(){
  const input=document.getElementById('broadcastInput');
  const content=input.value.trim();
  if(!content)return;
  await fetch(API+'/api/admin/broadcast',{method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify({content})});
  input.value='';
  alert('公告已发送');
}

// ===== 在线用户列表 =====
async function loadOnlineUsers(){
  try{
    const r=await fetch(API+'/api/online',{headers:{'Authorization':'Bearer '+token}});
    if(!r.ok)return;
    const users=await r.json();
    document.getElementById('onlineCount').textContent=users.length;
    const el=document.getElementById('onlineList');el.innerHTML='';
    users.forEach(u=>{
      const tag=document.createElement('span');
      tag.className='online-tag';
      tag.innerHTML='<span class="dot"></span>'+esc(u.username);
      el.appendChild(tag);
    });
  }catch(e){}
}

// ===== 修改密码 =====
function showChangePassword(){
  document.getElementById('oldPwd').value='';
  document.getElementById('newPwd').value='';
  document.getElementById('newPwd2').value='';
  document.getElementById('pwdError').textContent='';
  document.getElementById('pwdModal').classList.add('show');
}
function closeChangePassword(){
  document.getElementById('pwdModal').classList.remove('show');
}
async function doChangePassword(){
  const old=document.getElementById('oldPwd').value;
  const np=document.getElementById('newPwd').value;
  const np2=document.getElementById('newPwd2').value;
  const errEl=document.getElementById('pwdError');
  errEl.textContent='';
  if(!old||!np||!np2){errEl.textContent='请填写所有字段';return;}
  if(np!==np2){errEl.textContent='两次输入的新密码不一致';return;}
  if(np.length<8){errEl.textContent='新密码长度不能少于 8 位';return;}
  try{
    const r=await fetch(API+'/api/password',{method:'PUT',headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify({old_password:old,new_password:np})});
    const d=await r.json();
    if(r.ok){closeChangePassword();alert('密码修改成功，请重新登录');localStorage.removeItem('token');localStorage.removeItem('username');location.reload();}
    else{errEl.textContent=d.error||'修改失败';}
  }catch(e){errEl.textContent='网络错误';}
}

// ===== 消息分页（向上滚动加载更多） =====
let isLoadingMore=false,hasMoreMessages=true;
function setupMessageScroll(){
  const el=document.getElementById('messages');
  el.addEventListener('scroll',async()=>{
    if(el.scrollTop>50||isLoadingMore||!hasMoreMessages||!currentChannel)return;
    isLoadingMore=true;
    const firstMsg=el.querySelector('.msg-row .bubble');
    if(!firstMsg){isLoadingMore=false;return;}
    // 获取第一条消息的时间
    const msgs=el.querySelectorAll('.msg-row');
    if(msgs.length===0){isLoadingMore=false;return;}
    const loadMoreEl=document.createElement('div');
    loadMoreEl.className='load-more';
    loadMoreEl.innerHTML='<span class="loading-spinner"></span> 加载中...';
    el.insertBefore(loadMoreEl,el.firstChild);
    try{
      const r=await fetch(API+'/api/channels/'+currentChannel+'/messages?limit=30&before='+encodeURIComponent(firstMsgTime),{headers:{'Authorization':'Bearer '+token}});
      if(!r.ok){isLoadingMore=false;loadMoreEl.remove();return;}
      const olderMsgs=await r.json();
      if(olderMsgs.length===0){hasMoreMessages=false;loadMoreEl.textContent='没有更多消息了';setTimeout(()=>loadMoreEl.remove(),2000);isLoadingMore=false;return;}
      const prevHeight=el.scrollHeight;
      loadMoreEl.remove();
      if(olderMsgs.length>0)firstMsgTime=olderMsgs[0].CreatedAt;
      const fragment=document.createDocumentFragment();
      olderMsgs.forEach(m=>{
        if(shouldShowTime(m.CreatedAt)){
          const timeDiv=document.createElement('div');timeDiv.className='msg-time';timeDiv.innerHTML='<span>'+formatTime(m.CreatedAt)+'</span>';fragment.appendChild(timeDiv);
        }
        fragment.appendChild(createMsgElement(m.UserID,m.Username,m.Content,m.ID));
      });
      el.insertBefore(fragment,el.firstChild);
      el.scrollTop=el.scrollHeight-prevHeight;
      if(olderMsgs.length<30)hasMoreMessages=false;
    }catch(e){}
    isLoadingMore=false;
  });
}
let firstMsgTime='';
function createMsgElement(userId,username,content,msgId){
  const div=document.createElement('div');
  const isSelf=(username===myUsername);
  if(isSelf){
    div.className='msg-row self';
    div.innerHTML='<div class="avatar" style="background:'+avatarColor(username)+'">'+esc(username.charAt(0).toUpperCase())+'</div><div class="content"><div class="bubble" data-msgid="'+(msgId||'')+'">'+esc(content)+'</div></div>';
    if(msgId){
      let pressTimer;
      div.addEventListener('touchstart',()=>{pressTimer=setTimeout(()=>showMsgActions(div,msgId),500);});
      div.addEventListener('touchend',()=>clearTimeout(pressTimer));
      div.addEventListener('contextmenu',(e)=>{e.preventDefault();showMsgActions(div,msgId);});
    }
  }else{
    div.className='msg-row';
    div.innerHTML=avatarHTML(username)+'<div class="content"><div class="name">'+esc(username)+'</div><div class="bubble">'+esc(content)+'</div></div>';
  }
  return div;
}
function showMsgActions(rowEl,msgId){
  document.querySelectorAll('.msg-actions').forEach(el=>el.remove());
  const bubble=rowEl.querySelector('.bubble');
  const menu=document.createElement('div');
  menu.className='msg-actions';
  menu.innerHTML='<button onclick="editMsg(this,\''+esc(msgId)+'\')">编辑</button><button class="danger" onclick="deleteMsg(this,\''+esc(msgId)+'\')">删除</button>';
  bubble.style.position='relative';
  bubble.appendChild(menu);
  menu.style.display='block';
  setTimeout(()=>{
    const close=()=>{menu.remove();document.removeEventListener('click',close);};
    document.addEventListener('click',close);
  },0);
}

// ===== 消息编辑/删除 =====
async function editMsg(btn,msgId){
  const bubble=btn.closest('.bubble');
  const oldContent=bubble.textContent.replace('编辑删除','').trim();
  const newContent=prompt('编辑消息:',oldContent);
  if(!newContent||newContent===oldContent)return;
  try{
    const r=await fetch(API+'/api/messages/'+msgId,{method:'PUT',headers:{'Content-Type':'application/json','Authorization':'Bearer '+token},body:JSON.stringify({content:newContent})});
    if(r.ok){bubble.childNodes[0].textContent=newContent;}
    else{alert('编辑失败');}
  }catch(e){alert('编辑失败');}
}
async function deleteMsg(btn,msgId){
  if(!confirm('确定删除这条消息？'))return;
  try{
    const r=await fetch(API+'/api/messages/'+msgId,{method:'DELETE',headers:{'Authorization':'Bearer '+token}});
    if(r.ok){btn.closest('.msg-row').remove();}
    else{alert('删除失败');}
  }catch(e){alert('删除失败');}
}

// 页面加载时自动恢复登录状态
(async function(){
  const savedToken=localStorage.getItem('token');
  const savedUser=localStorage.getItem('username');
  if(savedToken&&savedUser){
    try{
      const r=await fetch(API+'/api/me',{headers:{'Authorization':'Bearer '+savedToken}});
      if(r.ok){token=savedToken;myUsername=savedUser;startApp();return;}
    }catch(e){}
    localStorage.removeItem('token');
    localStorage.removeItem('username');
  }
})();
</script>
</body></html>`
