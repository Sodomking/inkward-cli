const fs=require('node:fs'),path=require('node:path'),{execFileSync}=require('node:child_process');
const site=process.env.INKWARD_SITE_URL;
if(!site)throw new Error('Set INKWARD_SITE_URL to the deployed production origin first.');
require('./inkward.cjs').website({site});
fs.mkdirSync('release',{recursive:true});
const source=fs.readFileSync('inkward.cjs','utf8').replace('const RELEASE_SITE = null;',`const RELEASE_SITE = ${JSON.stringify(site)};`)
  .replace('if(require.main===module)main()', 'main()').replace('module.exports={options,website,main};','');
fs.writeFileSync('release/entry.cjs',source);
fs.writeFileSync('release/sea.json',JSON.stringify({main:'release/entry.cjs',output:'release/sea.blob',disableExperimentalSEAWarning:true,useSnapshot:false,useCodeCache:false}));
execFileSync(process.execPath,['--experimental-sea-config','release/sea.json'],{stdio:'inherit'});
const suffix=process.platform==='win32'?'.exe':'',name=`inkward-${process.platform}-${process.arch}${suffix}`,out=path.resolve('release',name);
fs.copyFileSync(process.execPath,out);
if(process.platform==='darwin'){
  // Universal Node binaries contain one SEA fuse per architecture. Thin before injection.
  const archs=execFileSync('lipo',['-archs',out],{encoding:'utf8'}).trim().split(/\s+/);
  if(archs.length>1)execFileSync('lipo',[out,'-thin',process.arch==='arm64'?'arm64':'x86_64','-output',out]);
  execFileSync('codesign',['--remove-signature',out]);
}
const postject=require.resolve('postject/dist/cli.js');
const args=[postject,out,'NODE_SEA_BLOB','release/sea.blob','--sentinel-fuse','NODE_SEA_FUSE_fce680ab2cc467b6e072b8b5df1996b2'];
if(process.platform==='darwin')args.push('--macho-segment-name','NODE_SEA');
execFileSync(process.execPath,args,{stdio:'inherit'});
if(process.platform==='darwin')execFileSync('codesign',['--sign','-',out]);
fs.chmodSync(out,0o755);
execFileSync(out,['info','--json'],{stdio:'inherit'});
const digest=require('node:crypto').createHash('sha256').update(fs.readFileSync(out)).digest('hex');
fs.writeFileSync(out+'.sha256',digest+'  '+name+'\n');
console.log('Built '+out);
