import { AuthService } from './auth.service';
import { ActivatedRouteStub } from '../util/common.test-util';
import { Observable } from 'rxjs/Observable';
import { ActivatedRoute } from '@angular/router';

declare var gapi: any;
let activatedRouteStub: ActivatedRoute;

describe('AuthService', () => {
  beforeEach(() => {
    activatedRouteStub = new ActivatedRoute();
    activatedRouteStub.queryParams = Observable.of({project: 'fakeProjectId'});
    this.authService = new AuthService(activatedRouteStub);
  });

  it('should fail loading user status if auth2 init fails', (async() => {
    gapi.initTestParams(false /* auth2InitSuccess */);

    this.authService.init();

    let loadAuthFailed = false;
    try {
      await this.authService.loadSignInStatus();
    } catch (e) {
      loadAuthFailed = true;
    }
    expect(loadAuthFailed).toBe(true);
    expect(this.authService.getCurrentUser()).toBeNull();
  }));

  it('should get current userinfo if user is already signed in', (async() => {
    gapi.initTestParams();
    this.authService.init();
    const signedIn = await this.authService.loadSignInStatus();
    expect(signedIn).toBe(true);
    expect(this.authService.getCurrentUser()).not.toBeNull();
    expect(this.authService.getCurrentUser().Name).toBe(gapi.USER_NAME);
  }));

  it('should return null user if no user signed in', (async() => {
    gapi.initTestParams(true /* auth2InitSuccess */,
                        false /* isSignedIn */);
    this.authService.init();
    const signedIn = await this.authService.loadSignInStatus();
    expect(signedIn).toBe(false);
    expect(this.authService.getCurrentUser()).toBeNull();
  }));

  it('should successfully sign in a user', (async() => {
    gapi.initTestParams(true /* auth2InitSuccess */,
                        false /* isSignedIn */,
                        true /* signInSuccess */);
    this.authService.init();
    const signedIn = await this.authService.loadSignInStatus();
    expect(signedIn).toBe(false);
    expect(this.authService.getCurrentUser()).toBeNull();

    await this.authService.signIn();
    expect(this.authService.getCurrentUser()).not.toBeNull();
    expect(this.authService.getCurrentUser().Name).toBe(gapi.USER_NAME);
  }));

  it('should not update user info if signing in a user fails', (async() => {
    gapi.initTestParams(true /* auth2InitSuccess */,
                        false /* isSignedIn */,
                        false /* signInSuccess */);
    this.authService.init();

    let failedSignIn = false;
    try {
     await this.authService.signIn();
    } catch (e) {
      failedSignIn = true;
    }
    expect(failedSignIn).toBe(true);
    expect(this.authService.getCurrentUser()).toBeNull();
  }));

  it('should sign out a user successfully', (async() => {
    gapi.initTestParams(true /* auth2InitSuccess */,
                        true /* isSignedIn */,
                        true /* signInSuccess */,
                        true /* signOutSuccess */);
    this.authService.init();
    const signedIn = await this.authService.loadSignInStatus();
    expect(signedIn).toBe(true);
    expect(this.authService.getCurrentUser()).not.toBeNull();
    expect(this.authService.getCurrentUser().Name).toBe(gapi.USER_NAME);

    await this.authService.signOut();
    expect(this.authService.getCurrentUser()).toBeNull();
  }));

  it('should not update user info if sign out fails', (async() => {
    gapi.initTestParams(true /* auth2InitSuccess */,
                        true /* isSignedIn */,
                        true /* signInSuccess */,
                        false /* signOutSuccess */);
    this.authService.init();

    const signedIn = await this.authService.loadSignInStatus();
    expect(signedIn).toBe(true);
    expect(this.authService.getCurrentUser()).not.toBeNull();

    let failedSignOut = false;
    try {
     await this.authService.signOut();
    } catch (e) {
      failedSignOut = true;
    }
    expect(failedSignOut).toBe(true);
    expect(this.authService.getCurrentUser()).not.toBeNull();
  }));
});
