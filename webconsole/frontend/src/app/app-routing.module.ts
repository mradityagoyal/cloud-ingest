import {AppComponent} from './app.component';
import {AuthService} from './auth/auth.service';
import {AgentComponent} from './agent/agent.component';
import {NgModule} from '@angular/core';
import {RouterModule, Routes } from '@angular/router';
import {BandwidthControlComponent} from './bandwidth-control/bandwidth-control.component';
import {PulseComponent} from './pulse/pulse.component';

const appRoutes: Routes = [
  {
    path: 'agent',
    component: AgentComponent,
    canActivate: [AuthService],
  },
  {
    path: 'bandwidth-control',
    component: BandwidthControlComponent,
    canActivate: [AuthService],
  },
  {
    path: 'pulse',
    component: PulseComponent
  }
];

@NgModule({
  imports: [
    RouterModule.forRoot(
      appRoutes
    )
  ],
  exports: [
    RouterModule
  ]
})
export class AppRoutingModule { }
