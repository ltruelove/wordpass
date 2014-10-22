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
                self.passwordKey(null);
                self.decryptedPasswords(data);
            },
            error: function(request, textStatus, errorThrown){
                alert("There was a problem fetching your passwords.");
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
                self.loggedInUser(data);
            },
            error: function(request, textStatus, errorThrown){
                alert("Username and password combination doesn't exist");
            }
        });

        self.logout = function(){
            self.user({Username: "", Password: ""});
            self.decryptedPasswords(null);
            self.loggedInUser(null);
            self.token = null;
        }
    };
}

$().ready(function(){
    ko.applyBindings(new indexViewModel());
});

