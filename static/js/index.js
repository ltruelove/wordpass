function indexViewModel() {
    var self = this;
    self.token = "";

    self.user = ko.observable({Username: "", Password: ""});

    self.login = function(){
        var userData = ko.toJS(self.user);
        var stringData = JSON.stringify(userData);

        $.ajax({
            cache: false,
            url: '/Login',
            type: 'POST',
            data: JSON.stringify(userData),
            contentType: "application/json",
            dataType: "json",
            success: function(data, textStatus, request){
                self.token = request.getResponseHeader("Token");
            },
            error: function(request, textStatus, errorThrown){
                alert("Username and password combination doesn't exist");
                /*
                console.log(errorThrown);
                console.log(textStatus);
                */
            }
        });

        Sammy(function() {
            this.get('', function() {
                this.app.runRoute('get', '#Login');
            });
        }).run();
    };
}

$().ready(function(){
    ko.applyBindings(new indexViewModel());
});

