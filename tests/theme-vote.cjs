const {chromium}=require('playwright');
const assert=require('node:assert/strict');
const {spawn}=require('node:child_process');
const net=require('node:net');

(async()=>{
 let server;
 let baseURL=process.env.TEST_BASE_URL;
 const errors=[];
 if(!baseURL){
  if(!process.env.TIER_DU_SERVER)throw new Error('Set TIER_DU_SERVER to a compiled server or TEST_BASE_URL to a running server.');
  const reservation=net.createServer();
  await new Promise(resolve=>reservation.listen(0,'127.0.0.1',resolve));
  const port=reservation.address().port;
  await new Promise(resolve=>reservation.close(resolve));
  server=spawn(process.env.TIER_DU_SERVER,[],{cwd:process.cwd(),env:{...process.env,PORT:String(port)},windowsHide:true,stdio:'pipe'});
  baseURL=`http://127.0.0.1:${port}`;
  const deadline=Date.now()+15000;
  while(true){try{if((await fetch(`${baseURL}/healthz`)).ok)break;}catch{}if(Date.now()>deadline)throw new Error('Test server did not start.');await new Promise(resolve=>setTimeout(resolve,100));}
 }
 const browser=await chromium.launch({...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{}),headless:true});
 const pages=[];
 try{
  for(let index=0;index<3;index++){
   const context=await browser.newContext(index?{viewport:{width:390,height:844},isMobile:true,hasTouch:true}:{viewport:{width:1100,height:850}});
   await context.route('https://fonts.googleapis.com/**',route=>route.abort());
   const page=await context.newPage();
   page.on('pageerror',error=>errors.push(error.message));
   page.on('websocket',socket=>socket.on('framereceived',frame=>{
    const message=JSON.parse(String(frame.payload));
    for(const field of ['themeVoteCounts','themeVotes','themeVotesNeeded'])if(Object.hasOwn(message,field))errors.push(`Private vote field exposed: ${field}`);
   }));
   await page.goto(baseURL);
   await page.locator('#inputName').fill(`Teste ${index}`);
   if(index===0)await page.locator('#btnCreate').click();
   else{
    const code=await pages[0].evaluate(()=>myState.code);
    await page.locator('#inputCode').fill(code);await page.locator('#btnJoin').tap();
   }
   await page.waitForFunction(()=>myState.phase==='lobby');
   pages.push(page);
  }
  const [host,second,third]=pages;
  await host.waitForFunction(()=>myState.players.length===3);
  await host.locator('#btnStart').click();
  for(const page of pages){await page.waitForFunction(()=>myState.phase==='instructions');await page.getByRole('button',{name:'Prosseguir com o jogo',exact:true}).click();}
  for(const page of pages)await page.waitForFunction(()=>myState.phase==='themeVote');
  const options=await host.evaluate(()=>myState.themeOptions);
  assert.equal(options.length,5);
  assert.equal(await host.locator('#btnConfirmThemeVote').isDisabled(),true);
  await host.locator('.theme-option').nth(0).click();
  assert.equal(await host.locator('.theme-option').nth(0).getAttribute('aria-pressed'),'true');
  assert.equal(await host.evaluate(()=>myState.myThemeVote),'');
  await host.locator('#btnConfirmThemeVote').click();
  await host.locator('#btnEditThemeVote').waitFor();
  assert.equal(await host.evaluate(()=>myState.myThemeVote),options[0]);
  assert.equal(await second.evaluate(()=>myState.myThemeVote),'');
  await host.locator('#btnEditThemeVote').click();
  await host.locator('#btnConfirmThemeVote').waitFor();
  await host.locator('.theme-option').nth(1).click();
  await second.locator('.theme-option').nth(1).tap();
  await second.locator('#btnConfirmThemeVote').tap();
  await second.locator('#btnEditThemeVote').waitFor();
  assert.equal(await host.evaluate(()=>myState.phase),'themeVote');
  assert.equal(await host.locator('.theme-option').nth(1).getAttribute('aria-pressed'),'true');
  await host.reload();
  await host.waitForFunction(()=>myState.phase==='themeVote'&&!awaitingState&&ws?.readyState===WebSocket.OPEN);
  assert.equal(await host.locator('.theme-option').nth(1).getAttribute('aria-pressed'),'true');
  assert.equal(await host.locator('#btnConfirmThemeVote').isEnabled(),true);
  await third.locator('.theme-option').nth(0).tap();
  await third.locator('#btnConfirmThemeVote').tap();
  await third.locator('#btnEditThemeVote').waitFor();
  assert.equal(await host.evaluate(()=>myState.phase),'themeVote');
  for(const page of pages){
   assert.doesNotMatch(await page.locator('.theme-options').innerText(),/\d+\s+votos/i);
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
  }
  await host.locator('#btnConfirmThemeVote').click();
  for(const page of pages){await page.waitForFunction(()=>myState.phase==='answering');assert.equal(await page.evaluate(()=>myState.currentTheme),options[1]);}
  assert.deepEqual(errors,[]);
  console.log('PASS: three real browser players; working clicks and mobile taps; selection before confirmation; private vote payloads and hidden counts; edit revokes confirmation; selection survives other votes and reload; reconfirmation chooses the updated winner; mobile layout.');
 }finally{
  for(const page of pages){try{await page.evaluate(()=>send('leave'));}catch{}}
  await browser.close();
  server?.kill();
 }
})().catch(error=>{console.error(error);process.exit(1);});
