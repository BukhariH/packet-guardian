// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
/* exported c */
var c = {
    flashTimeout: 0,

    FlashMessage: function(text, type) {
        var flash = $('#flashDiv'),
            flashClass = (type === 'success') ? 'flashSuccess' : 'flashFailure';

        // Post is a callback which is called after the message has faded out
        var clear = function(post) {
            post = (post) ? post : $.noop;
            flash.fadeOut(500, function() {
                flash.removeClass('flashSuccess');
                flash.removeClass('flashFailure');
                $('#flashText').html("");
                c.flashTimeout = 0;
                post();
            });
        };

        var show = function() {
            $('#flashText').html(text);
            flash.addClass(flashClass);
            flash.fadeIn(500);
            c.flashTimeout = setTimeout(clear, 10000);
        };

        if (c.flashTimeout) {
            clearTimeout(c.flashTimeout);
            clear(show);
            return;
        }
        show();
    },
};

$.onReady(function() {
    var flashMsg = $('#flashText').html();
    if (flashMsg !== '') {
        c.FlashMessage(flashMsg);
    }
});

function setSrcQuery(e, q) {
    var src = e.src;
    var p = src.indexOf('?');
    if (p >= 0) {
        src = src.substr(0, p);
    }
    e.src = src + "?" + q
}

function playAudio() {
    var e = document.getElementById('captchaAudio')
    setSrcQuery(e, "lang=en")
    e.style.display = 'block';
    e.autoplay = 'true';
    return false;
}

function reload() {
    setSrcQuery(document.getElementById('captchaImage'), "reload=" + (new Date()).getTime());
    setSrcQuery(document.getElementById('captchaAudio'), (new Date()).getTime());
    return false;
}
