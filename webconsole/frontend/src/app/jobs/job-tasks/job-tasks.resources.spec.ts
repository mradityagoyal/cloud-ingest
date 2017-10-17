import { FAKE_TASKS } from '../jobs.test-util';
import { fail } from 'assert';
import { async } from '@angular/core/testing';
import { SimpleDataSource } from './job-tasks.resources';
import { Task } from '../jobs.resources';

let simpleDataSource: SimpleDataSource;

describe('SimpleDataSource', () => {

    beforeEach(() => {
      simpleDataSource = new SimpleDataSource(FAKE_TASKS);
    });

    it('connect() should return the sample tasks', async(() => {
      simpleDataSource.connect().subscribe((tasks: Task[]) => {
        expect(tasks).toEqual(FAKE_TASKS);
      }, () => {
        fail('Should return the tasks.');
      });
    }));
});
