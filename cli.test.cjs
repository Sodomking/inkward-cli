const {test}=require('node:test'),assert=require('node:assert/strict');
const {options,website}=require('./inkward.cjs');
test('origin validation and invite encoding',()=>{
  const o=options(['open','--site','https://writing.example.com','--ref','friends_01','--exam','postgraduate']);
  assert.equal(website(o),'https://writing.example.com/?ref=friends_01#/exams?exam=postgraduate');
  for(const site of ['javascript:alert(1)','https://u:p@example.com','https://example.com/bad','http://example.com'])assert.throws(()=>website({site}));
  assert.match(website({site:'http://127.0.0.1:3000',dev:true}),/^http:/);
});
test('no implicit execution or arbitrary shell flags',()=>{
  assert.equal(options([]).command,'info');assert.equal(options(['open']).yes,false);
  assert.throws(()=>options(['--eval','bad']));assert.throws(()=>options(['--ref','a&bad']));assert.throws(()=>options(['--site']));
});
