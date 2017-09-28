import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'jobStatus'
})

export class JobStatusPipe implements PipeTransform {

  // TODO(b/64680346): Status name should be read from protocol buffer.
  transform(value: number): string {
    switch (value) {
      case 0: {
        return 'Not Started';
      }
      case 1: {
        return 'In Progress';
      }
      case 2: {
        return 'Failed';
      }
      case 3: {
        return 'Success';
      }
      default: {
        return 'Undefined';
      }
    }
  }
}
