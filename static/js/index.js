function indexViewModel() {
    var self = this;
    self.token = "";

    self.user = ko.observable({Username: "", Password: "", First: "", Last: "", PasswordKey:""});
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
                location.hash = "/userHome";
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
        location.hash = "/";
    };

    self.addBlank = function() {
        $.get('/Record/Blank', function(data) {
            if (data != null) {
                self.passwordContainer().decryptedPasswords.push(data);
            }
        }, 'json');
    };

    self.delete = function(record) {
        var i = self.passwordContainer().decryptedPasswords().indexOf(record);
        if(i != -1) {
            self.passwordContainer().decryptedPasswords.splice(i, 1);
            self.saveAll();
        }
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

    self.createUserAdmin = function(){
        if(!self.validateUser()){
            return false;
        }

        var user = self.user();

        $.ajax({
            cache: false,
            url: '/UserCreateAdmin',
            type: 'POST',
            data: JSON.stringify(user),
            contentType: "application/json",
            success: function(data, textStatus, request){
                self.login();
            },
            error: function(request, textStatus, errorThrown){
                alert("There was a problem creating the records");
            }
        });
    };

    self.validateUser = function(){
        var repeatInput = $('#repeatPassword');
        if(self.user().Password != repeatInput.val()){
            repeatInput.focus();
            alert("Passwords do not match");
            return false;
        }

        var keyRepeat = $('#repeatPasswordKey');
        if(self.user().PasswordKey != keyRepeat.val()){
            keyRepeat.focus();
            alert("Password keys do not match");
            return false;
        }

        return true;
    }
}

var model = new indexViewModel();

var app = Sammy('#main', function(){
    this.get('#/', function (context) {
        $.ajax({url: "/setupCheck",
            type: "GET",
            dataType: "json",
            success: function(data){
                if(data.IsNew){
                    context.app.runRoute('get', '#/setup');
                }else{
                    // if an existing user wasn't returned show the login page
                    if(!data.user){
                        context.partial('../home.html',null, function(){
                            var container = document.getElementById('login')
                            ko.cleanNode(container);
                            ko.applyBindings(model,container);
                        });
                    }else{ //go to the user home
                        context.app.runRoute('get', '#/userHome');
                    }
                }

            }
        });


    });

    this.get('#/setup', function(context) {
        context.partial('../setup.html',null, function(){
            var container = document.getElementById('setup')
            ko.cleanNode(container);
            ko.applyBindings(model,container);
        });
    });

    this.get('#/userHome', function (context) {
        context.partial('../userHome.html',null, function(){
            var container = document.getElementById('userHome')
            ko.cleanNode(container);
            ko.applyBindings(model,container);
        });
    });

    /*
    this.get('#/findCourt', function(context){
        context.partial('../findCourt.html',null, function(){
            var container = document.getElementById('findCourts')
            ko.cleanNode(container);
            ko.applyBindings(model,container);
        });
    });    
    */

});

$().ready(function(){
    app.run('#/');
});

