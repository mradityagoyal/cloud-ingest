import { ActivatedRoute } from '@angular/router';
import { ActivatedRouteStub } from '../util/common.test-util';
import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AgentComponent } from './agent.component';

let activatedRouteStub: ActivatedRouteStub;

describe('AgentComponent', () => {
  let component: AgentComponent;
  let fixture: ComponentFixture<AgentComponent>;

  beforeEach(() => {
    activatedRouteStub = new ActivatedRouteStub();
    TestBed.configureTestingModule({
      declarations: [
        AgentComponent
      ],
      imports: [
        AngularMaterialImporterModule,
        BrowserAnimationsModule
      ],
      providers: [
        {provide: ActivatedRoute, useValue: activatedRouteStub}
      ]
    }).compileComponents();
    fixture = TestBed.createComponent(AgentComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
