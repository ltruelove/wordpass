function indexViewModel() {
    var self = this;
    self.token = "";

    self.user = ko.observable({Username: "", Password: ""});
    self.loggedInUser = ko.observable();
    self.passwordKey = ko.observable();
    self.passwordContainer = ko.observable();

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
                //self.passwordKey(null);
                var decrypted = ko.observableArray(data);
                self.passwordContainer({decryptedPasswords:decrypted});
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
    };

    self.logout = function(){
        self.user({Username: "", Password: ""});
        self.passwordContainer(null);
        self.loggedInUser(null);
        self.token = null;
        self.passwordKey(null);
    };

    self.addBlank = function() {
        $.get('/Record/Blank', function(data) {
            if (data != null) {
                self.passwordContainer().decryptedPasswords.push(ko.observable(data));
            }
        }, 'json');
    };

    self.saveAll = function(){
        var userData = {PasswordKey: self.passwordKey(), Passwords: self.passwordContainer().decryptedPasswords()};
        var stringData = JSON.stringify(userData);

        if(self.passwordKey() == null || self.passwordKey() == ""){
            alert("You must enter your password key in order to save the records.");
            return;
        }

        $.ajax({
            cache: false,
            url: '/Records',
            type: 'POST',
            data: stringData,
            contentType: "application/json",
            success: function(data, textStatus, request){
                alert("Save was successful");
            },
            error: function(request, textStatus, errorThrown){
                alert("There was a problem saving the records");
            },
            beforeSend: function(request){
                request.setRequestHeader("Token", self.token);
            }
        });
    };
}

$().ready(function(){
    ko.applyBindings(new indexViewModel());
});

