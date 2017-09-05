/**
 * @fileoverview This file contains a module that imports all of the angular
 *     material (https://material.angular.io/) components used in this
 *     application. This way, all of the angular material imports are kept in
 *     one place.
 */
import { NgModule } from '@angular/core';
import { MdIconModule,
        MdSidenavModule,
        MdListModule,
        MdToolbarModule,
        MdCardModule,
        MdTooltipModule,
        MdButtonModule,
        MdDialogModule,
        MdInputModule,
        MdCheckboxModule } from '@angular/material';

@NgModule({
  imports: [MdIconModule,
            MdSidenavModule,
            MdListModule,
            MdToolbarModule,
            MdCardModule,
            MdTooltipModule,
            MdButtonModule,
            MdDialogModule,
            MdInputModule,
            MdCheckboxModule],
  exports: [MdIconModule,
            MdSidenavModule,
            MdListModule,
            MdToolbarModule,
            MdCardModule,
            MdTooltipModule,
            MdButtonModule,
            MdDialogModule,
            MdInputModule,
            MdCheckboxModule]
})

export class AngularMaterialImporterModule { }
