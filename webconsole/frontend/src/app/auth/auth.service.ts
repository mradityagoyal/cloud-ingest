import { Injectable } from '@angular/core';
import { CanActivate } from '@angular/router';
import { UserProfile } from './auth.resources';
import { environment } from '../../environments/environment';

/**
  * TODO(b/65052265): Implement type definitions for gapi library. Library documentations:
  *   https://developers.google.com/api-client-library/javascript/reference/referencedocs
  *   https://developers.google.com/identity/sign-in/web/reference
  */
declare let gapi: any;

const AUTH_SCOPE = 'https://www.googleapis.com/auth/cloud-platform';

/**
 * AuthService is responsible for authenticating a user using a Google Account.
 * https://apis.google.com/js/platform.js should be loaded to use this class.
 */
@Injectable()
export class AuthService implements CanActivate {
  private loadGapiPromise: Promise<void>;
  private gapiAuth2: any;
  private user: UserProfile = null;

  /**
   * Initialize the class and load the Google API (gapi). This method has to be
   * called before calling any other method in this class.
   */
  init() {
    this.loadGapiPromise = new Promise<void>((resolve, reject) => {
      gapi.load('auth2', {
        callback: () => {
          this.gapiAuth2 = gapi.auth2.init({
            client_id: environment.authClientId,
            scope: AUTH_SCOPE,
          });
          resolve();
        },
        onerror: () => {
          reject(new Error('Failed to init gapi.'));
        },
      });
    });
  }

  /**
   * Loads the current signed in user data if any.
   * @return{Promise<boolean>} Whether a user is already signed in.
   */
  loadSignInStatus(): Promise<boolean> {
    return new Promise<boolean>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;

        this.gapiAuth2.then(
            (auth) => {
               if (auth.isSignedIn.get()) {
                 this.user = <UserProfile>{
                   Name: auth.currentUser.get().getBasicProfile().getName(),
                 };
               }
               resolve(auth.isSignedIn.get());
            },
            (error) => {
               reject(error);
            });
      })();
    });
  }

  /**
   * Gets the current signed in user authorization header.
   * @return{string} The signed in user authorization header, null if user is
   *     not signed in.
   */
  getAuthorizationHeader(): string {
    if (this.user == null) {
      return null;
    }
    const auth =
        gapi.auth2.getAuthInstance().currentUser.get().getAuthResponse();
    return `Bearer ${auth.access_token}`;
  }

  /**
   * Implementation for auth guard deciding if a route can be activated.
   */
  canActivate(): Promise<boolean> {
    return this.loadSignInStatus();
  }

  /*
   * Get the current signed in user profile, null if no user signed in.
   */
  getCurrentUser(): UserProfile {
    return this.user;
  }

  /*
   * Sign out the current user.
   */
  signOut(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      this.loadGapiPromise.then(() => {
        this.gapiAuth2.signOut().then(
          () => {
            this.user = null;
            resolve();
          },
          (error) => {
            reject(error);
          });
      });
    });
  }

  /*
   * Sign in using Google account.
   */
  signIn(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      this.loadGapiPromise.then(() => {
        this.gapiAuth2.signIn({
                  prompt: 'select_account'
                })
                .then((user) => {
                  this.user = <UserProfile>{
                    Name: user.getBasicProfile().getName(),
                  };
                  resolve();
                },
                (error) => {
                  reject(error);
                });
      });
    });
  }
}
