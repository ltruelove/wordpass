function indexViewModel() {
    var self = this;
    self.token = "";

    self.user = ko.observable({Username: "", Password: ""});
    self.loggedInUser = ko.observable();
    self.passwordKey = ko.observable();
    self.decryptedPasswords = ko.observableArray();

    self.unlock = function(){
        var userData = {PasswordKey: self.passwordKey()};

        $.ajax({
            cache: false,
            url: '/RecordList',
            type: 'POST',
            data: JSON.stringify(userData),
            contentType: "application/json",
            dataType: "json",
            success: function(data, textStatus, request){
                console.log(data);
                self.passwordKey(null);
                self.decryptedPasswords(data);
            },
            error: function(request, textStatus, errorThrown){
                alert("There was a problem fetching your passwords.");
                /*
                console.log(errorThrown);
                console.log(textStatus);
                */
            },
            beforeSend: function(request){
                request.setRequestHeader("Token", self.token);
            }
        });
    };

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
                self.user(null);
                location.hash = '/home';
                console.log(data);
                self.loggedInUser(data);
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

