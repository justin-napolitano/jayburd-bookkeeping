import { test } from 'node:test';
import assert from 'node:assert/strict';
import { receipt } from './image-receipt.mjs';
const input = { project: 'synthetic-project', component: 'api', commit: 'a'.repeat(40), run: '1', attempt: '2', imageID: 'sha256:' + 'b'.repeat(64) };
const build = { 'containerimage.config.digest': input.imageID, 'buildx.build.provenance': { builder: { id: 'synthetic' } } };
const image = 'us-central1-docker.pkg.dev/synthetic-project/jayburd-dev/api@sha256:' + 'c'.repeat(64);
test('receipt identifies one published candidate, never grants deployment authority', () => {
 const value = receipt(input, build, [image]);
 assert.equal(value.image, image);
 assert.equal(value.deployable, false);
});
test('mismatched image, missing provenance, wrong repository and ambiguous digest are rejected', () => {
 assert.throws(() => receipt(input, {...build, 'containerimage.config.digest': 'sha256:' + 'd'.repeat(64)}, [image]));
 assert.throws(() => receipt(input, {'containerimage.config.digest': input.imageID}, [image]));
 assert.throws(() => receipt(input, build, [image.replace('/api@', '/web@')]));
 assert.throws(() => receipt(input, build, [image, image]));
 assert.throws(() => receipt({...input, commit: 'main'}, build, [image]));
});
