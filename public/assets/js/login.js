// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
$.onReady(function() {
    'use strict';
    var login = function() {
        $('#login-btn').prop('disabled', true);

        var data = {};
        data.username = $('[name=username]').value();
        data.password = $('[name=password]').value();

        if (data.username === '' || data.password === '') {
            return;
        }

        $('#login-btn').text("Logging in...");

        API.login(data, function() {
            location.href = '/';
        }, function(req) {
            $('#login-btn').text("Login >");
            $('#login-btn').prop('disabled', false);
            if (req.status === 401) {
                c.FlashMessage("Incorrect username or password");
            } else {
                c.FlashMessage("Unknown error");
            }
        });
    };

    $('#login-btn').click(function() {
        login();
    });

    $('[name=username]').keyup(function(e) {
        if (e.keyCode === 13) {
            login();
        }
    });

    $('[name=password]').keyup(function(e) {
        if (e.keyCode === 13) {
            login();
        }
    });
});
