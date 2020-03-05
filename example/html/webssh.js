
function checkwindow() {
	event.returnValue=false;
}

function atou(encodeString) {
  return decodeURIComponent(escape(atob(encodeString)));
}

function utoa(rawString) {
  return btoa(encodeURIComponent(rawString));
}

function readFile(element_id, res_id) {
    const objFile = document.getElementById(element_id);
	
    if(objFile.value === '') {
		//alert('请选择文件！');
		return
    }
	
    // 获取文件
    const files = objFile.files;
	
    // 新建一个FileReader
    const reader = new FileReader();
	
    // 读取文件 
    reader.readAsText(files[0], "UTF-8");
	
	// 读取完文件之后会回来这里
    reader.onload = function(e){
      // 读取文件内容
      var fileString = e.target.result;
	  
      // 接下来可对文件内容进行处理
	  $("#" + res_id).text(fileString);
    }
}

function get_term_size() {
    var init_width = 9;
    var init_height = 17;

    var windows_width = $(window).width();
    var windows_height = $(window).height();

    return {
        cols: Math.floor(windows_width / init_width),
        rows: Math.floor(windows_height / init_height),
    }
}

function get_connect_info() {
	var hostname = location.hostname;
	var protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';
	var ws_port = (location.port) ? (':' + location.port) : '';
	
    var host = $.trim($('#host').val());
    var port = $.trim($('#port').val());
    var user = $.trim($('#user').val());
    var auth = $("input[name='auth']:checked").val();
    var passwd = $.trim($('#password').val());
    var ssh_key = null;

    if (auth === 'key') {
		ssh_key = $("#pkey_res").text();
    }
	
	var cols_rows = get_term_size();
	
	return {
		hostname: hostname,
		protocol: protocol,
		ws_port: ws_port,
		host: host,
		port: port,
		user: user,
		auth: auth,
		passwd: passwd,
		ssh_key: ssh_key,
		cols: cols_rows.cols,
		rows: cols_rows.rows,
	}
}

function ws_connect() {
    var connect_info = get_connect_info();
	
    var term = new Terminal({
		cols: connect_info.cols,
		rows: connect_info.rows,
		useStyle: true,
		cursorBlink: true
	});
	
    var socketURL = connect_info.protocol + connect_info.hostname + connect_info.ws_port + '/api/ssh';

    var socket = new WebSocket(socketURL);
	
	socket.onopen = function () {
        $('#form').addClass('hide');
        $('#webssh-terminal').removeClass('hide');
        term.open(document.getElementById('terminal'));
		term.focus();
		$("body").attr("onbeforeunload",'checkwindow()'); //增加刷新关闭提示属性
		socket.send(JSON.stringify({ type: "addr", data: utoa(connect_info.host + ":" + connect_info.port) }));
		//socket.send(JSON.stringify({ type: "term", data: utoa("linux") }));
		socket.send(JSON.stringify({ type: "login", data: utoa(connect_info.user) }));
		if (connect_info.auth === 'pwd') {
			socket.send(JSON.stringify({ type: "password", data: utoa(connect_info.passwd) }));
		} else if (connect_info.auth === 'key') {
			socket.send(JSON.stringify({ type: "publickey", data: utoa(connect_info.ssh_key) }));
		};
		socket.send(JSON.stringify({ type: "resize", cols: connect_info.cols, rows: connect_info.rows }));
		
		// 发送数据
        term.on('data', function (data) {
			socket.send(JSON.stringify({ type: "stdin", data: btoa(data) }));
        });
		
		// 接收数据
        socket.onmessage = function (recv) {
			var msg = JSON.parse(recv.data);
			switch (msg.type) {
				case "stdout":
				case "stderr":
					term.write(atou(msg.data));
			}
        };
		
		// 连接错误
        socket.onerror = function (e) {
			term.write('connect error');
            console.log(e);
        };
		
		// 关闭连接
        socket.onclose = function (e) {
            console.log(e);
			term.write('disconnect');
            //term.destroy();
        };
    };
	
	// 监听浏览器窗口, 根据浏览器窗口大小修改终端大小
	$(window).resize(function () {
		var cols = get_term_size().cols;
		var rows = get_term_size().rows;
		socket.send(JSON.stringify({ type: "resize", cols: cols, rows: rows }));
		term.resize(cols, rows);
	});
}
