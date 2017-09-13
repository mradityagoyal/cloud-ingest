import { TestBed, async } from '@angular/core/testing';
import { JobsService } from './jobs.service';
import { JobConfig } from './api.resources';
import { JobConfigsComponent } from './job-configs.component';
import { AngularMaterialImporterModule } from './angular-material-importer.module';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';
import 'rxjs/add/operator/delay';
import 'rxjs/add/observable/never';

class JobsServiceStub {
  public getJobConfigs = jasmine.createSpy('getJobConfigs');
}

const FAKE_JOBSPEC1: any = {'fakeField1': 'fakeValue1', 'fakeField2' : 'fakeValue2'};
const FAKE_JOBSPEC2: any = {'fakeField3': 'fakeValue3', 'fakeField4' : 'fakeValue4'};
const FAKE_JOBSPEC3: any = {'fakeField5': 'fakeValue5', 'fakeField6' : 'fakeValue6'};

const FAKE_JOB_CONFIGS: JobConfig[] = [
  {
    JobConfigId: 'fakeJobConfigId1',
    JobSpec: FAKE_JOBSPEC1
  },
  {
    JobConfigId: 'fakeJobConfigId2',
    JobSpec: FAKE_JOBSPEC2
  },
  {
    JobConfigId: 'fakeJobConfigId3',
    JobSpec: FAKE_JOBSPEC3
  }
];

let jobsServiceStub: JobsServiceStub;

describe('JobConfigsComponent', () => {

  beforeEach(async(() => {
    jobsServiceStub = new JobsServiceStub();
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.of(FAKE_JOB_CONFIGS));

    TestBed.configureTestingModule({
      declarations: [
        JobConfigsComponent
      ],
      providers: [
        {provide: JobsService, useValue: jobsServiceStub},
      ],
      imports: [
        AngularMaterialImporterModule
      ],
    }).compileComponents();
  }));

  it('should create the job runs component', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  }));

  it('should initialize the component with show loading spinner as false', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component.showLoadingSpinner = false);
  }));

  it('should show a loading spinner while job configs are loading', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = true;
    jobsServiceStub.getJobConfigs.and.returnValue(Observable.never());
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('md-spinner');
      expect(element).not.toBeNull();
    });
  }));

  it('should not show a loading spinner when job configs return', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('md-spinner');
      expect(element).toBeNull();
    });
  }));

  it('should contain three md cards', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const elements = compiled.querySelectorAll('md-card');
      expect(elements.length).toBe(3);
    });
  }));

  it('should contain the job config information in cards', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const elements = compiled.querySelectorAll('md-card');
      expect(elements[0].innerText).toContain('fakeJobConfigId1');
      expect(elements[0].innerText).toContain('fakeField1');
      expect(elements[0].innerText).toContain('fakeValue1');
      expect(elements[0].innerText).toContain('fakeField2');
      expect(elements[0].innerText).toContain('fakeValue2');

      expect(elements[1].innerText).toContain('fakeJobConfigId2');
      expect(elements[1].innerText).toContain('fakeField3');
      expect(elements[1].innerText).toContain('fakeValue3');
      expect(elements[1].innerText).toContain('fakeField4');
      expect(elements[1].innerText).toContain('fakeValue4');

      expect(elements[2].innerText).toContain('fakeJobConfigId3');
      expect(elements[2].innerText).toContain('fakeField5');
      expect(elements[2].innerText).toContain('fakeValue5');
      expect(elements[2].innerText).toContain('fakeField6');
      expect(elements[2].innerText).toContain('fakeValue6');
    });
  }));

  it('should contain an add job configuration button', async(() => {
    const fixture = TestBed.createComponent(JobConfigsComponent);
    const component = fixture.debugElement.componentInstance;
    component.showLoadingSpinner = false;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.ingest-add-job-config');
      expect(element).not.toBeNull();
    });
  }));
});
