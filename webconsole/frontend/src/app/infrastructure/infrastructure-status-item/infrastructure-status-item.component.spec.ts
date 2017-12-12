import { async, TestBed } from '@angular/core/testing';

import { AngularMaterialImporterModule } from '../../angular-material-importer/angular-material-importer.module';
import { ResourceStatus } from '../../proto/tasks.js';
import { InfrastructureStatusItemComponent } from './infrastructure-status-item.component';

describe('InfrastructureStatusItemComponent', () => {

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [
        InfrastructureStatusItemComponent
      ],
      imports: [
        AngularMaterialImporterModule
      ],
    }).compileComponents();
  }));

  it('should create the component', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    expect(component).toBeTruthy();
  }));

  it('should contain an mat-list-item', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = ResourceStatus.Type.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelector('mat-list-item')).not.toBeNull();
    });
  }));

  it('should contain an mat-icon', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = ResourceStatus.Type.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelector('mat-icon')).not.toBeNull();
    });
  }));

  it('should contain the name title', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = ResourceStatus.Type.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.textContent).toContain('Fake Name Title');
    });
  }));

  it('should contain the status', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = ResourceStatus.Type.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.item-status');
      expect(element).not.toBeNull();
    });
  }));
});
