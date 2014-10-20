function indexViewModel() {
    var self = this;
    self.token = "";

    self.user = {Username: ko.observable(""), Password: ko.observable("")};

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
                /*
                console.log(errorThrown);
                console.log(textStatus);
                */
            }
        });
    };
}

$().ready(function(){
    ko.applyBindings(new indexViewModel());
});

