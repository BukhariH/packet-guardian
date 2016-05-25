// This source file is part of the Packet Guardian project.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
$.onReady(function() {
    $('[name=logout-btn]').click(function() {
        location.href = "/logout";
        return;
    });
    $('[name=admin-btn]').click(function() {
        location.href = "/admin";
        return;
    });
    $('[name=add-device-btn]').click(function(e) {
        isAdmin = $(e.target).data("admin");
        user = $('[name=username]').value();
        if (isAdmin !== null) {
            location.href = "/register?manual=1&username="+user;
        } else {
            location.href = "/register?manual=1";
        }
        return;
    });

    // Delete buttons
    $('[name=del-all-btn]').click(function() {
        var username = $('[name=username]').value();
        $.ajax({
            method: "DELETE",
            url: "/api/device/"+username,
            success: function() {
                location.reload();
            },
            error: function() {
                c.FlashMessage("Error deleting devices");
            }
        });
    });

    $('[name=del-selected-btn]').click(function() {
        var checked = $('.device-select:checked');
        if (checked.length === 0) {
            return;
        }

        var username = $('[name=username]').value();
        var devicesToRemove = [];
        for (var i = 0; i < checked.length; i++) {
            devicesToRemove.push(checked[i].value);
        }

        $.ajax({
            method: 'DELETE',
            url: '/api/device/'+username,
            params: {"mac": devicesToRemove.join(',')},
            success: function() {
                location.reload();
            },
            error: function() {
                c.FlashMessage("Error deleting devices");
            }
        });
    });

    c.BindSelectAll('[name=dev-sel-all]', '.device-select');
});
