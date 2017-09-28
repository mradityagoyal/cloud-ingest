import { JobStatusPipe } from './job-status.pipe';

describe('JobStatusPipe', () => {
  it('create an instance', () => {
    const pipe = new JobStatusPipe();
    expect(pipe).toBeTruthy();
  });

  it('correctly return not started', () => {
    const pipe = new JobStatusPipe();
    const result = pipe.transform(0);
    expect(result).toEqual('Not Started');
  });

  it('correctly return in progress', () => {
    const pipe = new JobStatusPipe();
    const result = pipe.transform(1);
    expect(result).toEqual('In Progress');
  });

  it('correctly return failed', () => {
    const pipe = new JobStatusPipe();
    const result = pipe.transform(2);
    expect(result).toEqual('Failed');
  });

  it('correctly return success', () => {
    const pipe = new JobStatusPipe();
    const result = pipe.transform(3);
    expect(result).toEqual('Success');
  });

  it('handle unknown statuses', () => {
    const pipe = new JobStatusPipe();
    const result = pipe.transform(984);
    expect(result).toEqual('Undefined');
  });
});
