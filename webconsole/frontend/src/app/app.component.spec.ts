import { TestBed, async } from '@angular/core/testing';
import { ActivatedRoute, Params, NavigationExtras } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { FormsModule } from '@angular/forms';
import { Observable } from 'rxjs/Observable';
import { AppComponent } from './app.component';
import { AngularMaterialImporterModule } from './angular-material-importer.module';
import { AuthService } from './auth.service';
import { UserProfile } from './auth.resources';
import { NoopAnimationsModule} from '@angular/platform-browser/animations';
import 'rxjs/add/observable/of';

const FAKE_USER = 'Fake User';
const FAKE_AUTH = 'Fake Auth';

class MockAuthService extends AuthService {
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
}

let activatedRouteStub: ActivatedRoute;

describe('AppComponent', () => {
  const mockAuthService = new MockAuthService();

  beforeEach(async(() => {
    mockAuthService.isSignedIn = true;
    activatedRouteStub = new ActivatedRoute();
    activatedRouteStub.queryParams = Observable.of({project: 'fakeProjectId'});

    TestBed.configureTestingModule({
      declarations: [
        AppComponent
      ],
      providers: [
        {provide: AuthService, useValue: mockAuthService},
        {provide: ActivatedRoute, useValue: activatedRouteStub},
      ],
      imports: [
        RouterTestingModule,
        NoopAnimationsModule,
        FormsModule,
        AngularMaterialImporterModule
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
          toContain(`Ingest Web Console - ${FAKE_USER}`);
    });
  }));

  it('should contain four links and signout', async(() => {
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

  it('should contain a Job Runs link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#jobrunslink');
      expect(element).not.toBeNull();
      expect(element.textContent).toContain('Job Runs');
      expect(element.getAttribute('queryParamsHandling')).toBe('merge');
    });
  }));

  it('should contain a Job Configs link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#jobconfigslink');
      expect(element).not.toBeNull();
      expect(element.textContent).toContain('Job Configs');
      expect(element.getAttribute('queryParamsHandling')).toBe('merge');
    });
  }));

  it('should contain a Create Job Run link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('#createjobrunlink');
      expect(element).not.toBeNull();
      expect(element.textContent).toContain('Create Job Run');
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
      const element = compiled.querySelector('md-toolbar');
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
      const element = compiled.querySelector('md-sidenav');
      expect(element).not.toBeNull();
    });
  }));

  it('should navigate to the job configs page with project id', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    app.gcsProjectId = 'fakeGcsProjectId';
    const navigateSpy = spyOn((<any>app).router, 'navigate');
    const fakeNavigationExtras: NavigationExtras = {
      queryParams: { project: 'fakeGcsProjectId' }
    };

    app.onProjectSelectSubmit();
    expect(navigateSpy).toHaveBeenCalledWith(['/jobconfigs'], fakeNavigationExtras);
  }));

  it('should show a project selection input if no project is selected', async(() => {
    activatedRouteStub.queryParams = Observable.of({project: ''});
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const projectInput = compiled.querySelector('#projectId');
      expect(projectInput).not.toBeNull();
    });
  }));

});
