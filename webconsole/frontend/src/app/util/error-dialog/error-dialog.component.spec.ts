import { ErrorDialogContent } from './error-dialog.resources';
import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';

import { ErrorDialogComponent } from './error-dialog.component';

import { MatDialogRefStub } from '../../util/common.test-util';

let matDialogRefStub: MatDialogRefStub;
const FAKE_ERROR_CONTENT: ErrorDialogContent = {
  errorTitle: 'fakeErrorTitle',
  errorMessage: 'fakeErrorMessage'
};

describe('ErrorDialogComponent', () => {
  let component: ErrorDialogComponent;
  let fixture: ComponentFixture<ErrorDialogComponent>;

  beforeEach(async(() => {
    matDialogRefStub = new MatDialogRefStub();

    TestBed.configureTestingModule({
      declarations: [ ErrorDialogComponent ],
      providers: [
        {provide: MatDialogRef, useValue: matDialogRefStub},
        {provide: MAT_DIALOG_DATA, useValue: FAKE_ERROR_CONTENT}
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ErrorDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should contain the error title and message', () => {
    const compiled = fixture.debugElement.nativeElement;
    expect(compiled.textContent).toContain('fakeErrorTitle');
    expect(compiled.textContent).toContain('fakeErrorMessage');
  });

});
