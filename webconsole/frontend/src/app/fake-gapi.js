/**
  * Define the fake gapi namespace. For more documentation, see
  *  https://developers.google.com/api-client-library/javascript/reference/referencedocs
  */
var gapi = {};

gapi.USER_NAME = 'Fake User';

// Initialize the test params for faking gapi.
gapi.initTestParams = function(auth2InitSuccess = true,
                               isSignedIn = true,
                               signInSuccess = true,
                               signOutSuccess = true,
                               userName = gapi.USER_NAME) {
  gapi.testConfig = {};
  gapi.testConfig.auth2InitSuccess = auth2InitSuccess;
  gapi.testConfig.isSignedIn = isSignedIn;
  gapi.testConfig.signInSuccess = signInSuccess;
  gapi.testConfig.signOutSuccess = signOutSuccess;
  gapi.testConfig.userName = userName;
};

gapi.load = function(components, params) {
    params.callback();
};

/**
  * Define the fake gapi.auth2 namespace. For more documentation, see
  * https://developers.google.com/identity/sign-in/web/reference
  */
gapi.auth2 = {};

gapi.auth2.init = function(param){
  return new gapi.auth2.GoogleAuth();
};

// Fake GoogleAuth class implementation.
gapi.auth2.GoogleAuth = class {
  constructor() {
    this.isSignedIn = new gapi.auth2.IsSignedIn();
    this.currentUser = new gapi.auth2.CurrentUser();
  }

  then(callback, error) {
    if (gapi.testConfig.auth2InitSuccess) {
      callback(this);
    } else {
      error(new Error('Fake auth init error'));
    }
  }

  signIn() {
    return new gapi.auth2.SignInPromise();
  }

  signOut() {
    return new gapi.auth2.SignOutPromise();
  }
};

// Fake IsSignedIn class implementation.
gapi.auth2.IsSignedIn = class {
  get() {
    return gapi.testConfig.isSignedIn;
  }
};

// Fake Current class implementation.
gapi.auth2.CurrentUser = class {
  get() {
    return new gapi.auth2.GoogleUser();
  }
};

// Fake GoogleUser class implementation.
gapi.auth2.GoogleUser = class {
  getBasicProfile() {
    return new gapi.auth2.BasicProfile();
  }
};

// Fake BasicProfile class implementation.
gapi.auth2.BasicProfile = class {
  getName() {
    return gapi.testConfig.userName;
  }
};

// Fake SignInPromise class implementation.
gapi.auth2.SignInPromise = class {
  then(callback, error) {
    if (gapi.testConfig.signInSuccess) {
      callback(new gapi.auth2.GoogleUser());
    } else {
      error(new Error('Failed sigining in'));
    }
  }
};

// Fake SignOutPromise class implementation.
gapi.auth2.SignOutPromise = class {
  then(callback, error) {
    if (gapi.testConfig.signOutSuccess) {
      callback();
    } else {
      error(new Error('Failed sigining out'));
    }
  }
};

