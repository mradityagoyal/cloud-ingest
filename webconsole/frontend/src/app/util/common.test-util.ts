import { AuthService } from '../auth/auth.service';
import { UserProfile } from '../auth/auth.resources';

export class MatDialogStub {
  public open = jasmine.createSpy('open');
}

export class ActivatedRouteStub {
  snapshot = {
    paramMap: {
      get: jasmine.createSpy('get')
    },
    queryParams: {
      project: 'fakeProjectId'
    }
  };
  queryParams = jasmine.createSpy('queryParams');
}

export class MatSnackBarStub {
  open = jasmine.createSpy('open');
}

export const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};

export class MatDialogRefStub {
  public close = jasmine.createSpy('close');
  public afterClosed = jasmine.createSpy('afterClosed');
}

export class AuthServiceStub {
  public init = jasmine.createSpy('init');
  public loadSignInStatus = jasmine.createSpy('loadSignInStatus');
  public getAuthorizationHeader = jasmine.createSpy('getAuthorizationHeader');
  public getCurrentUser = jasmine.createSpy('getCurrentUser');
  public grantPubsubTopicPermissionsIfNotExists = jasmine.createSpy('grantPubsubTopicPermissionsIfNotExists');
  public grantBucketPermissionsIfNotExist = jasmine.createSpy('grantBucketPermissions');
}

export const FAKE_USER = 'Fake User';
export const FAKE_AUTH = 'Fake Auth';

export class MockAuthService extends AuthService {
  isSignedIn = true;

  fakeUser: UserProfile = {
    Name: FAKE_USER
  };

  init() { }

  loadSignInStatus(): Promise<boolean> {
    return Promise.resolve(this.isSignedIn);
  }

  getAuthorizationHeader(): string {
    return FAKE_AUTH;
  }

  getCurrentUser(): UserProfile  {
    return this.fakeUser;
  }

  grantPubsubTopicPermissionsIfNotExists(): Promise<boolean> {
    return Promise.resolve(true);
  }

  grantBucketPermissionsIfNotExist(): Promise<boolean> {
    return Promise.resolve(true);
  }
}
