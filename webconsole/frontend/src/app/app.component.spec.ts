import { HttpClientTestingModule } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { MatSnackBar } from '@angular/material';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ActivatedRoute } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { Observable, of } from 'rxjs';

import { AngularMaterialImporterModule } from './angular-material-importer/angular-material-importer.module';
import { AppComponent } from './app.component';
import { AuthService } from './auth/auth.service';
import { ProjectSelectComponent } from './project-select/project-select.component';
import { ProjectSelectModule } from './project-select/project-select.module';
import { FAKE_USER, MatSnackBarStub, MockAuthService } from './util/common.test-util';



let activatedRouteStub: ActivatedRoute;
let matSnackBarStub: MatSnackBarStub;
let mockAuthService: MockAuthService;

describe('AppComponent', () => {

  beforeEach(async(() => {
    activatedRouteStub = new ActivatedRoute();
    activatedRouteStub.queryParams = of({project: 'fakeProjectId'});
    mockAuthService = new MockAuthService(activatedRouteStub);
    mockAuthService.isSignedIn = true;
    matSnackBarStub = new MatSnackBarStub();

    TestBed.configureTestingModule({
      declarations: [
        AppComponent,
        ProjectSelectComponent
      ],
      providers: [
        {provide: AuthService, useValue: mockAuthService},
        {provide: ActivatedRoute, useValue: activatedRouteStub},
        {provide: MatSnackBar, useValue: matSnackBarStub},
      ],
      imports: [
        RouterTestingModule,
        NoopAnimationsModule,
        ProjectSelectModule,
        AngularMaterialImporterModule,
        HttpClientTestingModule
      ],
    }).compileComponents();
  }));

  it('should create the app', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    expect(app).toBeTruthy();
  }));

  it('should render title in a h1 tag', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelector('h1').textContent).
          toContain(`On-Premises Transfer Service Web Console - ${FAKE_USER}`);
    });
  }));

  it('should contain three links and signout', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelectorAll('a').length).toBe(3);

      const signOutButton = compiled.querySelector('button');
      expect(signOutButton).not.toBeNull();
      expect(signOutButton.textContent).toContain('Signout');
    });
  }));

  it('should contain a Jobs link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#jobslink');
      expect(element).not.toBeNull();
      expect(element.textContent).toContain('Jobs');
      expect(element.getAttribute('queryParamsHandling')).toBe('merge');
    });
  }));

  it('should not show links, and show sign in button if not signed in',
     async(() => {
    mockAuthService.isSignedIn = false;
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelectorAll('a').length).toBe(0);
      const signInButton = compiled.querySelector('button');
      expect(signInButton).not.toBeNull();
      expect(signInButton.textContent).toContain('Sign In');
    });
  }));

  it('should contain a toolbar', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-toolbar');
      expect(element).not.toBeNull();
    });
  }));

  it('should contain a side nav', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('mat-sidenav');
      expect(element).not.toBeNull();
    });
  }));

  it('should show a project selection component if no project is selected', async(() => {
    activatedRouteStub.queryParams = of({project: ''});
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const projectInput = compiled.querySelector('app-project-select');
      expect(projectInput).not.toBeNull();
    });
  }));

  it('should open a snackbar on sign in failure', async(() => {
    spyOn(mockAuthService, 'signIn').and.returnValue(Promise.reject({error: 'fakeSignInFailedMessage'}));
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    app.signIn();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(matSnackBarStub.open).toHaveBeenCalled();
      expect(matSnackBarStub.open.calls.first().args[0]).toMatch('fakeSignInFailedMessage');
    });
  }));

  it('should open a snackbar on sign out failure', async(() => {
    spyOn(mockAuthService, 'signOut').and.returnValue(Promise.reject({error: 'fakeSignOutFailedMessage'}));
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    app.signOut();
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(matSnackBarStub.open).toHaveBeenCalled();
      expect(matSnackBarStub.open.calls.first().args[0]).toMatch('fakeSignOutFailedMessage');
    });
  }));

  it('should open a snackbar on loadSignInStatus failure when initiating the component', async(() => {
    spyOn(mockAuthService, 'loadSignInStatus').and.returnValue(Promise.reject({error: 'fakeLoadSignInFailMessage'}));
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      expect(matSnackBarStub.open).toHaveBeenCalled();
      expect(matSnackBarStub.open.calls.first().args[0]).toMatch('fakeLoadSignInFailMessage');
    });
  }));
});
