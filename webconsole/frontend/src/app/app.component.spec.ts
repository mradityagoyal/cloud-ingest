import { TestBed, async } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { AppComponent } from './app.component';
import { AngularMaterialImporterModule } from './angular-material-importer.module';

describe('AppComponent', () => {
  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [
        AppComponent
      ],
      imports: [
        RouterTestingModule,
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
    const compiled = fixture.debugElement.nativeElement;
    expect(compiled.querySelector('h1').textContent).
        toContain('Ingest Web Console');
  }));

  it('should contain four links', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    expect(compiled.querySelectorAll('a').length).toBe(4);
  }));

  it('should contain a Job Runs link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const element = compiled.querySelector('#jobrunslink');
    expect(element).not.toBeNull();
    expect(element.textContent).toContain('Job Runs');
  }));

  it('should contain a Job Configs link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const element = compiled.querySelector('#jobconfigslink');
    expect(element).not.toBeNull();
    expect(element.textContent).toContain('Job Configs');
  }));

  it('should contain a Create Job Config link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const element = compiled.querySelector('#createconfiglink');
    expect(element).not.toBeNull();
    expect(element.textContent).toContain('Create Job Config');
  }));

  it('should contain a Create Job Run link', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const element = compiled.querySelector('#createjobrunlink');
    expect(element).not.toBeNull();
    expect(element.textContent).toContain('Create Job Run');
  }));

  it('should contain a toolbar', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const element = compiled.querySelector('md-toolbar');
    expect(element).not.toBeNull();
  }));

  it('should contain a side nav', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.debugElement.nativeElement;
    const element = compiled.querySelector('md-sidenav');
    expect(element).not.toBeNull();
  }));

});
