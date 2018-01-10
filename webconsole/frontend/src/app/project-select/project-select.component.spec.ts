import { async, ComponentFixture, TestBed, inject } from '@angular/core/testing';
import { HttpClient } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { NavigationExtras } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { ProjectSelectComponent } from './project-select.component';
import { ProjectSelectModule } from './project-select.module';
import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { GoogleCloudApiProjectsResponse, GoogleCloudProject } from './project-select.resources';
import 'rxjs/add/operator/elementAt';

const FAKE_API_RESPONSE: GoogleCloudApiProjectsResponse = {
  projects: [
    {name: 'fakeName1', projectId: 'fakeId1', projectNumber: 1111111111},
    {name: 'fakeName2', projectId: 'fakeId2', projectNumber: 2222222222},
    {name: 'differentName', projectId: 'fakeId3', projectNumber: 3333333333}
  ]
};

const FAKE_API_RESPONSE_TOKEN: GoogleCloudApiProjectsResponse = {
  projects: [
    {name: 'fakeProject1', projectId: 'fakeProjectId1', projectNumber: 4444444444},
    {name: 'fakeProject2', projectId: 'fakeProjectId2', projectNumber: 5555555555},
    {name: 'fakeProject3', projectId: 'fakeProjectId3', projectNumber: 6666666666}
  ],
  nextPageToken: 'fakeProjectToken'
};

const FAKE_API_RESPONSE_TOKEN2: GoogleCloudApiProjectsResponse = {
  projects: [
    {name: 'fakeProject4', projectId: 'fakeProjectId4', projectNumber: 7777777777},
    {name: 'fakeProject5', projectId: 'fakeProjectId5', projectNumber: 8888888888},
    {name: 'fakeProject6', projectId: 'fakeProjectId6', projectNumber: 9999999999}
  ]
};

describe('ProjectSelectComponent', () => {
  let component: ProjectSelectComponent;
  let fixture: ComponentFixture<ProjectSelectComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ProjectSelectComponent ],
      imports: [ ProjectSelectModule,
                 AngularMaterialImporterModule,
                 HttpClientTestingModule,
                 RouterTestingModule ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ProjectSelectComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should be created', () => {
    expect(component).toBeTruthy();
  });

  it('should request the gcp resource manager get projects url', () => {
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
      httpMock.expectOne(ProjectSelectComponent.GCP_RESOURCE_MANAGER_API_URL).flush(FAKE_API_RESPONSE);
      expect(component.googleCloudProjects).toEqual(FAKE_API_RESPONSE.projects);
      httpMock.verify();
    });
  });

  it('should show the filtered projects', async(
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
      httpMock.expectOne(ProjectSelectComponent.GCP_RESOURCE_MANAGER_API_URL).flush(FAKE_API_RESPONSE);
      httpMock.verify();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        component.gcsProjectIdControl.setValue('differentN');
        fixture.detectChanges();
        fixture.whenStable().then(() => {
          fixture.detectChanges();
          // Take the second emit of the filtered projects. The first one is just the initial one.
          component.filteredGoogleCloudProjects.elementAt(1).subscribe((projects: GoogleCloudProject[]) => {
            expect(projects).toContain(FAKE_API_RESPONSE.projects[2]);
            expect(projects).not.toContain(FAKE_API_RESPONSE.projects[0]);
            expect(projects).not.toContain(FAKE_API_RESPONSE.projects[1]);
          });
        });
      });
    })
  ));

  it('should request the projects until there is no token', async(
    inject([HttpClient, HttpTestingController], (http: HttpClient, httpMock: HttpTestingController) => {
      httpMock.expectOne(ProjectSelectComponent.GCP_RESOURCE_MANAGER_API_URL).flush(FAKE_API_RESPONSE_TOKEN);
      httpMock.expectOne(ProjectSelectComponent.GCP_RESOURCE_MANAGER_API_URL + '?pageToken=fakeProjectToken')
          .flush(FAKE_API_RESPONSE_TOKEN2);
      httpMock.verify();
      fixture.detectChanges();
      fixture.whenStable().then(() => {
        expect(component.googleCloudProjects).toContain(FAKE_API_RESPONSE_TOKEN.projects[0]);
        expect(component.googleCloudProjects).toContain(FAKE_API_RESPONSE_TOKEN.projects[1]);
        expect(component.googleCloudProjects).toContain(FAKE_API_RESPONSE_TOKEN.projects[2]);
        expect(component.googleCloudProjects).toContain(FAKE_API_RESPONSE_TOKEN2.projects[0]);
        expect(component.googleCloudProjects).toContain(FAKE_API_RESPONSE_TOKEN2.projects[1]);
        expect(component.googleCloudProjects).toContain(FAKE_API_RESPONSE_TOKEN2.projects[2]);
      });
    })
  ));


  it('should navigate to the job configs page with project id in input', async(() => {
    component.gcsProjectIdControl.setValue('fakeGcsProjectId');
    const navigateSpy = spyOn((<any>component).router, 'navigate');
    const fakeNavigationExtras: NavigationExtras = {
      queryParams: { project: 'fakeGcsProjectId' }
    };
    component.onProjectSelectSubmit();
    expect(navigateSpy).toHaveBeenCalledWith(['/jobs'], fakeNavigationExtras);
  }));

});
