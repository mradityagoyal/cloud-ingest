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
        MdCheckboxModule,
        MdTableModule,
        MdProgressSpinnerModule,
        MdTabsModule,
        MdSnackBarModule,
        MdAutocompleteModule } from '@angular/material';

import { CdkTableModule } from '@angular/cdk/table';

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
            MdCheckboxModule,
            MdTableModule,
            MdProgressSpinnerModule,
            CdkTableModule,
            MdTabsModule,
            MdAutocompleteModule],
  exports: [MdIconModule,
            MdSidenavModule,
            MdListModule,
            MdToolbarModule,
            MdCardModule,
            MdTooltipModule,
            MdButtonModule,
            MdDialogModule,
            MdInputModule,
            MdCheckboxModule,
            MdTableModule,
            MdProgressSpinnerModule,
            MdTabsModule,
            MdSnackBarModule,
            CdkTableModule,
            MdAutocompleteModule]
})

export class AngularMaterialImporterModule { }
