import { Injectable } from '@angular/core';
import { CanActivate } from '@angular/router';
import { UserProfile, Policy, UpdatePolicyRequest, PolicyResponse } from './auth.resources';
import { environment } from '../../environments/environment';
import { ActivatedRoute, Params } from '@angular/router';
import { HttpErrorResponse, HttpResponse } from '@angular/common/http';

/**
  * TODO(b/65052265): Implement type definitions for gapi library. Library documentations:
  *   https://developers.google.com/api-client-library/javascript/reference/referencedocs
  *   https://developers.google.com/identity/sign-in/web/reference
  */
declare let gapi: any;

const PUBSUB_TOPICS = [];
const AUTH_SCOPE = 'https://www.googleapis.com/auth/cloud-platform';
const PUBSUB_SCOPE = 'https://www.googleapis.com/auth/pubsub';
const PROJECT_TEMPLATE_FIELD = '${project}';
const BUCKET_TEMPLATE_FIELD = '${bucket}';
const SET_PROJECT_POLICY_URL = 'https://cloudresourcemanager.googleapis.com/v1/projects/' + PROJECT_TEMPLATE_FIELD + ':setIamPolicy';
const GET_PROJECT_POLICY_URL = 'https://cloudresourcemanager.googleapis.com/v1/projects/' + PROJECT_TEMPLATE_FIELD + ':getIamPolicy';
const BUCKET_POLICY_URL = 'https://www.googleapis.com/storage/v1/b/' + BUCKET_TEMPLATE_FIELD + '/iam';
const ROBOT_ACCOUNT = environment.robotAccountEmail;
const ROBOT_SERVICE_ACCOUNT = 'serviceAccount:' + ROBOT_ACCOUNT;
const PUBSUB_EDITOR_ROLE = 'roles/pubsub.editor';
// TODO: Restrict output metadata to a separate bucket, or restrict to object
// reader and implement a more resilient approach in the DCP.
const STORAGE_OBJECT_CREATOR_ROLE = 'roles/storage.objectAdmin';

/**
 * AuthService is responsible for authenticating a user using a Google Account.
 * https://apis.google.com/js/platform.js should be loaded to use this class.
 */
@Injectable()
export class AuthService implements CanActivate {
  private loadGapiPromise: Promise<void>;
  private gapiAuth2: any;
  private gapiPubSub: any;
  private user: UserProfile = null;
  private projectId: string;

  constructor(private route: ActivatedRoute) {
    route.queryParams.subscribe(
      (params: Params) => {
        this.projectId = params.project;
      }
    );
  }

  /**
   * Initialize the class and load the Google API (gapi). This method has to be
   * called before calling any other method in this class.
   */
  init() {
    this.loadGapiPromise = new Promise<void>((resolve, reject) => {
      gapi.load('client:auth2', {
        callback: () => {
          this.gapiAuth2 = gapi.auth2.init({
            client_id: environment.authClientId,
            scope: AUTH_SCOPE + ' ' +
                   PUBSUB_SCOPE,
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

  private policyHasRobotAccountInRole(policy: Policy, role: string): boolean {
    for (const binding of policy.bindings) {
      if (binding.role === role) {
        if (binding.members.includes(ROBOT_SERVICE_ACCOUNT)) {
          return true;
        }
      }
    }
    return false;
  }

  private getProjectIamPolicy(projectId: string): Promise<PolicyResponse> {
    const url = GET_PROJECT_POLICY_URL.replace(PROJECT_TEMPLATE_FIELD, projectId);
    return new Promise<PolicyResponse>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;
        gapi.client.request({
          path: url,
          method: 'POST',
          body: {}
        }).then(
          (response: PolicyResponse) => {
             resolve(response);
          },
          (error: HttpErrorResponse) => {
             reject(error);
          });
      })();
    });
  }

  private getBucketIamPolicy(bucketName: string): Promise<PolicyResponse> {
    const url = BUCKET_POLICY_URL.replace(BUCKET_TEMPLATE_FIELD, bucketName);
    return new Promise<PolicyResponse>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;
        gapi.client.request({
          path: url,
          method: 'GET',
        }).then(
          (response: PolicyResponse) => {
             resolve(response);
          },
          (error: HttpErrorResponse) => {
             reject(error);
          });
      })();
    });
  }

  /**
   * Adds the robot account permissions to an existing policy.
   *
   * @param policy The current policy.
   */
  private addRobotAccountToRole(policy: Policy, iamRole: string): Policy {
    const newPolicy: Policy = {
      bindings: policy.bindings
    };
    let hasRole = false;
    for (const binding of newPolicy.bindings) {
      if (binding.role === iamRole) {
        binding.members.push(ROBOT_SERVICE_ACCOUNT);
        hasRole = true;
      }
    }
    if (!hasRole) {
      newPolicy.bindings.push({
        role: iamRole,
        members: [
          ROBOT_SERVICE_ACCOUNT
        ]
      });
    }
    return newPolicy;
  }

  private grantPubSubTopicPermissions(projectId: string, policy: Policy): Promise<Policy> {
    const newPolicy: Policy = this.addRobotAccountToRole(policy, PUBSUB_EDITOR_ROLE);
    return new Promise<Policy>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;
        const url = SET_PROJECT_POLICY_URL.replace(PROJECT_TEMPLATE_FIELD, projectId);
        gapi.client.request({
          path: url,
          method: 'POST',
          body: {
            policy: newPolicy,
          },
        }).then(
          (response: Policy) => {
             resolve(response);
          },
          (error: HttpErrorResponse) => {
             reject(error);
          });
      })();
    });
  }

  private grantBucketPermissions(bucketName: string, policy: Policy): Promise<Policy> {
    const newPolicy: Policy = this.addRobotAccountToRole(policy, STORAGE_OBJECT_CREATOR_ROLE);
    return new Promise<Policy>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;
        const url = BUCKET_POLICY_URL.replace(BUCKET_TEMPLATE_FIELD, bucketName);
        gapi.client.request({
          path: url,
          method: 'PUT',
          body: newPolicy,
        }).then(
          (response: Policy) => {
             resolve(response);
          },
          (error: HttpErrorResponse) => {
             reject(error);
          });
      })();
    });
  }

  /**
   * Grant writing permissions to the gcs bucket.
   * @param bucketName the bucket to grant permissions to.
   */
  grantBucketPermissionsIfNotExist(bucketName: string): Promise<boolean> {
    return new Promise<boolean>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;
        this.getBucketIamPolicy(bucketName).then(
          (policyResponse: PolicyResponse) => {
            const policy: Policy = policyResponse.result;
            if (this.policyHasRobotAccountInRole(policy, STORAGE_OBJECT_CREATOR_ROLE)) {
              resolve(true);
            } else {
              this.grantBucketPermissions(bucketName, policy);
              resolve(true);
            }
          },
          (error: HttpErrorResponse) => {
            reject(error);
          });
      })();
    });
  }

  /**
   * Idempotently grants roles/pubsub.editor role to the service account on the
   * input project id.
   *
   * @param projectId project to grant permissions to.
   */
  grantPubsubTopicPermissionsIfNotExists(): Promise<boolean> {
    return new Promise<boolean>((resolve, reject) => {
      (async() => {
        await this.loadGapiPromise;
        this.getProjectIamPolicy(this.projectId).then(
          (policyResponse: PolicyResponse) => {
            const policy: Policy = policyResponse.result;
            if (this.policyHasRobotAccountInRole(policy, PUBSUB_EDITOR_ROLE)) {
              resolve(true);
            } else {
              this.grantPubSubTopicPermissions(this.projectId, policy);
              resolve(true);
            }
          },
          (error: HttpErrorResponse) => {
            reject(error);
          });
      })();
    });
  }

}
