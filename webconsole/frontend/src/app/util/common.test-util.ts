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
}

export class MatSnackBarStub {
  open = jasmine.createSpy('open');
}

export const FAKE_HTTP_ERROR = {error : {error: 'FakeError', message: 'Fake Error Message.'}};

export class MatDialogRefStub {
  public close = jasmine.createSpy('close');
  public afterClosed = jasmine.createSpy('afterClosed');
}

