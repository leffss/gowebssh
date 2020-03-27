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
		let fileString = e.target.result;
		// 接下来可对文件内容进行处理
		$("#" + res_id).text(fileString);
    }
}

function get_term_size() {
    let init_width = 9;
	let init_height = 17;
	let windows_width = $(window).width();
	let windows_height = $(window).height();
    return {
		cols: Math.floor(windows_width / init_width),
		rows: Math.floor(windows_height / init_height),
    }
}

function get_connect_info() {
	let hostname = location.hostname;
	let protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';
	let ws_port = (location.port) ? (':' + location.port) : '';

	let host = $.trim($('#host').val());
	let port = $.trim($('#port').val());
	let user = $.trim($('#user').val());
	let auth = $("input[name='auth']:checked").val();
	let passwd = $.trim($('#password').val());
	let ssh_key = null;

    if (auth === 'key') {
		ssh_key = $("#pkey_res").text();
    }

	let cols_rows = get_term_size();
	
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
    let connect_info = get_connect_info();

	Terminal.applyAddon(attach);
	Terminal.applyAddon(fit);
	Terminal.applyAddon(fullscreen);
	Terminal.applyAddon(search);
	Terminal.applyAddon(terminado);
	Terminal.applyAddon(webLinks);
	Terminal.applyAddon(zmodem);

    let term = new Terminal({
		cols: connect_info.cols,
		rows: connect_info.rows,
		useStyle: true,
		cursorBlink: true
	});

    let socket = new WebSocket(connect_info.protocol + connect_info.hostname + connect_info.ws_port + '/api/ssh');

	function uploadFile(zsession) {
        let uploadHtml = "<div>" +
            "<label class='upload-area' style='width:100%;text-align:center;' for='fupload'>" +
            "<input id='fupload' name='fupload' type='file' style='display:none;' multiple='true'>" +
            "<i class='fa fa-cloud-upload fa-3x'></i>" +
            "<br />" +
            "点击选择文件" +
            "</label>" +
            "<br />" +
            "<span style='margin-left:5px !important;' id='fileList'></span>" +
            "</div><div class='clearfix'></div>";

        let upload_dialog = bootbox.dialog({
            message: uploadHtml,
            title: "上传文件",
            buttons: {
				cancel: {
					label: '关闭',
					className: 'btn-default',
					callback: function (res) {
						try {
							term.detach();
						} catch (e) {
							// console.log(e);
						}

						try {
							term.attach(socket);
						} catch (e) {
							// console.log(e);
						}

						try {
							// zsession 每 5s 发送一个 ZACK 包，5s 后会出现提示最后一个包是 ”ZACK“ 无法正常关闭
							// 这里直接设置 _last_header_name 为 ZRINIT，就可以强制关闭了
							zsession._last_header_name = "ZRINIT";
							zsession.close();
						} catch (e) {
							console.log(e);
						}
					}
				},
            },
			closeButton: false,
        });

        function hideModal() {
			upload_dialog.modal('hide');
		}

		let file_el = document.getElementById("fupload");

		return new Promise((res) => {
			file_el.onchange = function (e) {
				let files_obj = file_el.files;
				hideModal();
				Zmodem.Browser.send_files(zsession, files_obj, {
						on_offer_response(obj, xfer) {
							if (xfer) {
								term.write("\r\n");
							} else {
								term.write("\r\n" + obj.name + " was upload skipped");
								socket.send(JSON.stringify({ type: "ignore", data: utoa("\r\n" + obj.name + " was upload skipped\r\n") }));
							}
						},
						on_progress(obj, xfer) {
							updateProgress(xfer);
						},
						on_file_complete(obj) {
							socket.send(JSON.stringify({ type: "ignore", data: utoa("\r\n" + obj.name + " was upload success\r\n") }));
							// console.log("COMPLETE", obj);
						},
					}
				).then(zsession.close.bind(zsession), console.error.bind(console)
				).then(() => {
					res();
					term.write("\r\n");
				});
			};
		});
    }

	function saveFile(xfer, buffer) {
		return Zmodem.Browser.save_to_disk(buffer, xfer.get_details().name);
	}

	function updateProgress(xfer) {
		let detail = xfer.get_details();
		let name = detail.name;
		let total = detail.size;
		let percent;
		if (total === 0) {
			percent = 100
		} else {
			percent = Math.round(xfer._file_offset / total * 100);
		}

		term.write("\r" + name + ": " + total + " " + xfer._file_offset + " " + percent + "%    ");
	}

	function downloadFile(zsession) {
		zsession.on("offer", function(xfer) {
			function on_form_submit() {
				let FILE_BUFFER = [];
				xfer.on("input", (payload) => {
					updateProgress(xfer);
					FILE_BUFFER.push( new Uint8Array(payload) );
				});

				xfer.accept().then(
					() => {
						saveFile(xfer, FILE_BUFFER);
						term.write("\r\n");
						socket.send(JSON.stringify({ type: "ignore", data: utoa("\r\n" + xfer.get_details().name + " was download success\r\n") }));
					},
					console.error.bind(console)
				);
			}

			on_form_submit();

		});

		let promise = new Promise( (res) => {
			zsession.on("session_end", () => {
				res();
			});
		});

		zsession.start();
		return promise;
	}

	$('#form').addClass('hide');
	$('#webssh-terminal').removeClass('hide');
	term.open(document.getElementById('terminal'));
	term.focus();
	term.toggleFullScreen(true);
	$("body").attr("onbeforeunload",'checkwindow()'); //增加刷新关闭提示属性
	
	// 处理 zmodem
	term.on("zmodemDetect", (detection) => {
		try {
			term.detach();
		} catch (e) {
			// console.log(e);
		}
		let zsession = detection.confirm();
		let promise;
		if (zsession.type === "receive") {
			promise = downloadFile(zsession);
		} else {
			promise = uploadFile(zsession);
		}

		promise.catch( console.error.bind(console) ).then( () => {
			try {
				term.detach();
			} catch (e) {
				// console.log(e);
			}

			try {
				term.attach(socket);
			} catch (e) {
				// console.log(e);
			}
		});
	});
	
	socket.onopen = function () {
		socket.send(JSON.stringify({ type: "addr", data: utoa(connect_info.host + ":" + connect_info.port) }));

		//socket.send(JSON.stringify({ type: "term", data: utoa("linux") }));

		socket.send(JSON.stringify({ type: "login", data: utoa(connect_info.user) }));

		if (connect_info.auth === 'pwd') {
			socket.send(JSON.stringify({ type: "password", data: utoa(connect_info.passwd) }));
		} else if (connect_info.auth === 'key') {
			socket.send(JSON.stringify({ type: "publickey", data: utoa(connect_info.ssh_key) }));
		}

		socket.send(JSON.stringify({ type: "resize", cols: connect_info.cols, rows: connect_info.rows }));
		
		// 发送数据
        //term.on('data', function (data) {
		//	socket.send(JSON.stringify({ type: "stdin", data: btoa(data) }));
        //});

		// 接收数据
        //socket.onmessage = function (recv) {
		//	try {
		//		var msg = JSON.parse(recv.data);
		//		switch (msg.type) {
		//			case "stdout":
		//			case "stderr":
		//				term.write(atou(msg.data));
		//		}
		//	} catch (error) {
		//		console.log('接收的数据格式错误');
		//		console.log(recv.data);
		//	}
        //};

		//连接 term 和 socket
		//这种方式下 xterm.js 会把输入数据原样发送到后端，也会直接显示后端传来的数据
		term.attach(socket);
		term._initialized = true;
		term.zmodemAttach(socket, {
			noTerminalWriteOutsideSession: true,
		});
		
		// 连接错误
        socket.onerror = function (e) {
			term.write('connect error');
            console.log(e);
        };
		
		// 关闭连接
        socket.onclose = function (e) {
            console.log(e);
			term.write('disconnect');
			
			term.detach();
            //term.destroy();
        };
    };
	
	// 监听浏览器窗口, 根据浏览器窗口大小修改终端大小
	$(window).resize(function () {
		let cols_rows = get_term_size();
		socket.send(JSON.stringify({ type: "resize", cols: cols_rows.cols, rows: cols_rows.rows }));
		term.resize(cols_rows.cols, cols_rows.rows);
	});
}