const {chromium}=require('playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const http=require('node:http');
const path=require('node:path');
const os=require('node:os');

(async()=>{
 const html=fs.readFileSync('static/index.html');
 const server=http.createServer((req,res)=>{res.setHeader('Content-Type','text/html; charset=utf-8');res.end(html);});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 const browser=await chromium.launch({...(process.env.CHROME_PATH?{executablePath:process.env.CHROME_PATH}:{}),headless:true});
 try {
  const page=await browser.newPage({viewport:{width:1100,height:900}});
  const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.route('https://fonts.googleapis.com/**',r=>r.abort());
  await page.addInitScript(()=>{
   window.sent=[];
   window.WebSocket=class {
    static OPEN=1;static CONNECTING=0;
    constructor(){this.readyState=0;setTimeout(()=>{this.readyState=1;this.onopen?.();},0);}
    send(raw){const msg=JSON.parse(raw);window.sent.push(msg);if(msg.type==='join')this.onmessage?.({data:JSON.stringify({type:'joined',code:'TEST',name:'Host',host:true,token:'sample-token'})});}
    close(){this.readyState=3;}
   };
  });
  await page.goto(`http://127.0.0.1:${server.address().port}`);
  await page.locator('#inputName').fill('Host');await page.locator('#btnCreate').click();
  await page.waitForFunction(()=>session?.token==='sample-token');
  const base={code:'TEST',host:true,myID:'h',totalRounds:3,round:0,revision:0,players:[{id:'h',name:'Host',connected:true,host:true},{id:'a',name:'Ana',connected:true},{id:'b',name:'Bia',connected:true}],currentTheme:'Bebidas',board:[]};
  await page.evaluate(d=>{awaitingState=false;renderState({...d,phase:'instructions',readyCount:0,readyNeeded:3,iReady:false});},base);
  assert.equal(await page.locator('#leaveToolbar').isHidden(),true);
  assert.equal(await page.locator('.rules-card').count(),8);
  await page.locator('#btnReady').click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).type),'ready');
  await page.evaluate(d=>renderState({...d,phase:'instructions',readyCount:2,readyNeeded:3,iReady:true}),base);
  assert.equal(await page.locator('#btnReady').isDisabled(),true);
  const answering={...base,phase:'answering',round:1,revision:1,myTier:'C',myLeaderName:'Ana',timerEnd:Date.now()+120000,themeVotes:0,themeVotesNeeded:2,iAnswered:false};
  await page.evaluate(d=>renderState(d),answering);
  assert.equal(await page.locator('#leaveToolbar').isVisible(),true);
  assert.match(await page.locator('.tier-row.c').innerText(),/Mais ou menos/);
  for(const tier of ['S','A','B','C','E','F'])assert.match(await page.locator(`.tier-row.${tier.toLowerCase()} .tier-label`).innerText(),/\(.+\)/);
  assert.match(await page.locator('#timer').innerText(),/^(2:00|1:59)$/);
  assert.equal(await page.getByText('Trocar tema agora',{exact:true}).count(),0);
  await page.getByRole('button',{name:'🗳️ Votar para trocar tema'}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).type),'changeTheme');
  await page.locator('#answerText').fill('Café com leite');await page.locator('#btnAnswer').click();
  assert.deepEqual(await page.evaluate(()=>window.sent.at(-1)),{type:'answer',data:{text:'Café com leite'}});
  await page.evaluate(d=>renderState({...d,iAnswered:true,myAnswer:'Café com leite'}),answering);
  assert.equal(await page.locator('.submitted-answer p').innerText(),'Café com leite');
  await page.getByRole('button',{name:'Editar resposta'}).click();
  assert.equal(await page.evaluate(()=>window.sent.at(-1).type),'editAnswer');
  await page.evaluate(d=>renderState({...d,iAnswered:false,myAnswer:'Café com leite'}),answering);
  assert.equal(await page.locator('#answerText').inputValue(),'Café com leite');
  const answers=[{playerId:'h',playerName:'Host',text:'Minha resposta',original:'S',placed:'F',score:-7,likeCount:2,canLike:false},{playerId:'a',playerName:'Ana',text:'Resposta da Ana',original:'B',placed:'B',score:3,likeCount:3,canLike:true},{playerId:'b',playerName:'Bia',text:'Resposta da Bia',original:'C',placed:'E',score:1,likeCount:1,canLike:true}];
  const review={...base,phase:'review',round:3,nextRound:false,review:{round:3,theme:'Bebidas',score:-3,answers}};
  await page.evaluate(d=>renderState(d),review);
  assert.equal(await page.locator('#leaveToolbar').isHidden(),true);
  assert.equal(await page.getByRole('button',{name:'Sua própria resposta de Host'}).isDisabled(),true);
  assert.equal(await page.locator('.review-row').first().locator('strong.negative').innerText(),'-7');
  await page.getByRole('button',{name:'Curtir resposta de Ana'}).click();
  assert.deepEqual(await page.evaluate(()=>window.sent.at(-1)),{type:'like',data:{round:3,playerID:'a'}});
  const results=[{theme:'Bebidas',score:-3,answers},{theme:'Vilões',score:9,answers},{theme:'Matérias escolares',score:5,answers}];
  const stats=[{playerId:'h',playerName:'Host',score:-21,powerScore:0,powerUses:0},{playerId:'a',playerName:'Ana',score:9,powerScore:4,powerUses:2},{playerId:'b',playerName:'Bia',score:3,powerScore:-2,powerUses:1}];
  await page.evaluate(d=>renderState(d),{...base,phase:'final',round:3,results,stats});
  for(const title of ['🏆 Pódio dos temas','🎯 Mestres das respostas','🧠 Gênios das alterações','❤️ As 5 respostas mais curtidas'])assert.equal(await page.getByRole('heading',{name:title}).count(),1);
  assert.equal(await page.locator('.podium-place').count(),3);
  assert.equal(await page.locator('.popular-answer').count(),5);
  assert.equal(await page.locator('.all-results .review-row').count(),9);
  assert.equal(await page.locator('.negative-bar').count(),2);
  assert.equal(await page.locator('#leaveToolbar').isVisible(),true);
  const shotDir=process.env.UI_SCREENSHOT_DIR||os.tmpdir();fs.mkdirSync(shotDir,{recursive:true});
  const shot=path.join(shotDir,'tier-du-final-desktop.png');await page.screenshot({path:shot,fullPage:true});
  await page.setViewportSize({width:390,height:844});
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
  await page.screenshot({path:path.join(shotDir,'tier-du-final-mobile.png'),fullPage:true});
  await page.locator('#btnLeave').click();
  assert.equal(await page.evaluate(()=>localStorage.getItem('tier-du-session-v1')),null);
  assert.equal(await page.evaluate(()=>session),null);
  assert.equal(await page.evaluate(()=>window.sent.at(-1).type),'leave');
  assert.equal(await page.locator('#entry').evaluate(el=>el.classList.contains('active')),true);
  assert.deepEqual(errors,[]);
  console.log('PASS: instructions/ready, notes, 2:00, majority UI, edit/reconfirm messages, no own likes, progressive score, all four statistics, top five, mobile, explicit leave.');
  console.log('Screenshots: '+shot+'; '+path.join(shotDir,'tier-du-final-mobile.png'));
 }finally{await browser.close();await new Promise(resolve=>server.close(resolve));}
})().catch(e=>{console.error(e);process.exit(1)});
