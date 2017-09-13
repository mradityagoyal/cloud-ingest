import { TestBed, async } from '@angular/core/testing';
import { InfrastructureStatusItemComponent } from './infrastructure-status-item.component';
import { AngularMaterialImporterModule } from './angular-material-importer.module';
import { INFRA_STATUS } from './infrastructure.service';

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

  it('should contain an md-list-item', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = INFRA_STATUS.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelector('md-list-item')).not.toBeNull();
    });
  }));

  it('should contain an md-icon', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = INFRA_STATUS.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      expect(compiled.querySelector('md-icon')).not.toBeNull();
    });
  }));

  it('should contain the name title', async(() => {
    const fixture = TestBed.createComponent(InfrastructureStatusItemComponent);
    const component = fixture.debugElement.componentInstance;
    component.itemName = 'Fake Name Title';
    component.itemStatus = INFRA_STATUS.RUNNING;
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
    component.itemStatus = INFRA_STATUS.RUNNING;
    fixture.detectChanges();
    fixture.whenStable().then(() => {
      fixture.detectChanges();
      const compiled = fixture.debugElement.nativeElement;
      const element = compiled.querySelector('.item-status');
      expect(element).not.toBeNull();
    });
  }));
});
