const {chromium}=require('playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const http=require('node:http');
const os=require('node:os');
const path=require('node:path');

(async()=>{
 const server=http.createServer((request,response)=>{response.setHeader('Content-Type','text/html; charset=utf-8');response.end(fs.readFileSync('static/index.html'));});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 const browser=await chromium.launch({...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{}),headless:true});
 try {
  const context=await browser.newContext({viewport:{width:1100,height:900}});
  await context.route('https://fonts.googleapis.com/**',route=>route.abort());
  await context.addInitScript(()=>{
   window.sent=[];
   window.WebSocket=class {
    static OPEN=1;static CONNECTING=0;
    constructor(){this.readyState=0;setTimeout(()=>{this.readyState=1;this.onopen?.();},0);}
    send(raw){const message=JSON.parse(raw);window.sent.push(message);if(message.type==='join')this.onmessage?.({data:JSON.stringify({type:'joined',code:'TEST',name:'Host',host:true,token:'private-test-token'})});}
    close(){this.readyState=3;}
   };
  });
  const page=await context.newPage();
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto(`http://127.0.0.1:${server.address().port}`);
  await page.locator('#inputName').fill('Host');await page.locator('#btnCreate').click();
  await page.waitForFunction(()=>session?.token==='private-test-token');
  const base={code:'TEST',host:true,myID:'h',totalRounds:3,round:1,revision:1,players:[{id:'h',name:'Host',connected:true,host:true},{id:'a',name:'Ana',connected:true},{id:'b',name:'Bia',connected:true}],board:[]};
  const voting={...base,phase:'voting',themeOptions:['Frutas','Bebidas','Melhores emojis','Vilões','<img src=x onerror=alert(1)>'].map(theme=>({theme,votes:0})),votesCount:0,votesNeeded:3,myThemeVote:''};
  await page.evaluate(state=>{awaitingState=false;renderState(state);},voting);
  assert.equal(await page.locator('.theme-option').count(),5);
  assert.equal(await page.locator('.theme-option img').count(),0);
  await page.locator('.theme-option').nth(1).click();
  assert.deepEqual(await page.evaluate(()=>window.sent.at(-1)),{type:'voteTheme',data:{theme:'Bebidas',round:1}});
  await page.evaluate(state=>renderState({...state,myThemeVote:'Bebidas',votesCount:1}),voting);
  assert.equal(await page.locator('.theme-option[aria-pressed="true"]').count(),1);
  assert.match(await page.locator('[role="status"]').last().innerText(),/1 de 3/);
  const answering={...base,phase:'answering',currentTheme:'Bebidas',myTier:'E',myLeaderName:'Ana',timerEnd:Date.now()+120000,iAnswered:false};
  await page.evaluate(state=>renderState(state),answering);
  const secret=page.locator('#secretTier');
  assert.equal(await secret.innerText(),'?');
  const maskedSize=await secret.boundingBox();
  await secret.hover();await page.mouse.down();
  assert.match(await secret.innerText(),/^E/);
  assert.equal((await secret.boundingBox()).width,maskedSize.width);
  assert.equal((await secret.boundingBox()).height,maskedSize.height);
  await page.mouse.up();assert.equal(await secret.innerText(),'?');
  await secret.focus();await page.keyboard.down('Space');assert.match(await secret.innerText(),/^E/);
  await page.keyboard.up('Space');assert.equal(await secret.innerText(),'?');
  await secret.hover();await page.mouse.down();await page.evaluate(()=>window.dispatchEvent(new Event('blur')));
  assert.equal(await secret.innerText(),'?');await page.mouse.up();
  assert.equal(await page.getByRole('button',{name:/trocar tema/i}).count(),0);
  await page.locator('#answerText').fill('Café');
  await page.evaluate(state=>renderState(state),answering);
  assert.equal(await secret.innerText(),'?');assert.equal(await page.locator('#answerText').inputValue(),'Café');
  const power={...base,phase:'power',currentTheme:'Bebidas',discussingText:'Café',placed:'C',myUsedPower:true,myPowerChosen:false,iAmTarget:false,timerEnd:Date.now()+15000};
  await page.evaluate(state=>renderState(state),power);
  assert.equal(await page.locator('.power-options').count(),0);
  assert.match(await page.locator('.discussing-notice').innerText(),/Esperando resposta dos outros jogadores/);
  assert.match(await page.locator('#timer').innerText(),/^0:(15|14)$/);
  await page.evaluate(state=>renderState({...state,myUsedPower:false}),power);
  assert.equal(await page.locator('.power-options button').count(),3);
  for(const property of ['myPowerChosen','iAmTarget']){
   await page.evaluate(({state,property})=>renderState({...state,myUsedPower:false,[property]:true}),{state:power,property});
   assert.equal(await page.locator('.power-options').count(),0);
  }
  const result={round:1,theme:'Bebidas',score:1,answers:[{playerId:'a',playerName:'Ana',text:'Café',original:'S',placed:'A',score:1,likeCount:0,canLike:true}]};
  await page.evaluate(({base,result})=>renderState({...base,phase:'review',review:result,nextRound:true}),{base,result});
  assert.match(await page.locator('.review-row .grade').nth(0).innerText(),/Nota original/);
  assert.match(await page.locator('.review-row .grade').nth(1).innerText(),/Nota escolhida/);
  assert.ok((await page.locator('.review-row .grade').nth(0).boundingBox()).x<(await page.locator('.review-row .grade').nth(1).boundingBox()).x);
  await page.evaluate(state=>renderState({...state,phase:'changes',changes:[{playerName:'Ana',choice:1,before:'C',after:'B'}]}),power);
  assert.match(await page.locator('.review-row .grade').nth(0).innerText(),/Antes/);
  assert.match(await page.locator('.review-row .grade').nth(1).innerText(),/Depois/);
  await page.evaluate(({base,result})=>renderState({...base,phase:'final',results:[result],stats:[]}),{base,result});
  await page.locator('#btnRestart').click();
  assert.deepEqual(await page.evaluate(()=>window.sent.at(-1)),{type:'restart'});
  assert.equal(await page.evaluate(()=>session.token),'private-test-token');
  await page.evaluate(state=>renderState({...state,phase:'lobby',round:0,canStart:true}),base);
  assert.equal(await page.locator('#lobby').evaluate(element=>element.classList.contains('active')),true);
  assert.equal(await page.locator('.player-row').count(),3);
  assert.equal(await page.evaluate(()=>session.code),'TEST');
  await page.setViewportSize({width:390,height:844});
  const startBox=await page.locator('#btnStart').boundingBox(),leaveBox=await page.locator('#lobby > .kick').boundingBox();
  assert.ok(leaveBox.y-startBox.y-startBox.height>=24);
  await page.evaluate(state=>renderState(state),voting);
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
  await page.screenshot({path:path.join(os.tmpdir(),'tier-du-voting-mobile.png'),fullPage:true});
  const touchContext=await browser.newContext({viewport:{width:390,height:844},isMobile:true,hasTouch:true});
  await touchContext.route('https://fonts.googleapis.com/**',route=>route.abort());
  const touchPage=await touchContext.newPage();
  await touchPage.goto(`http://127.0.0.1:${server.address().port}`);
  await touchPage.evaluate(state=>renderState(state),answering);
  const touchSecret=touchPage.locator('#secretTier');
  await touchSecret.scrollIntoViewIfNeeded();
  const box=await touchSecret.boundingBox();
  const cdp=await touchContext.newCDPSession(touchPage);
  await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x:box.x+box.width/2,y:box.y+box.height/2}]});
  assert.match(await touchSecret.innerText(),/^E/);
  await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
  assert.equal(await touchSecret.innerText(),'?');
  await touchPage.screenshot({path:path.join(os.tmpdir(),'tier-du-secret-mobile.png'),fullPage:true});
  assert.deepEqual(errors,[]);
  console.log('PASS: five voting options, vote message, count, XSS, secret mouse/keyboard/touch hold and release, blur, fixed size, automatic power waiting, 15s, comparison order, restart keeps session and players, mobile spacing and overflow.');
 } finally {await browser.close();await new Promise(resolve=>server.close(resolve));}
})().catch(error=>{console.error(error);process.exit(1);});
