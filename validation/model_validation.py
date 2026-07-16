from __future__ import annotations
import csv, json, math, random
from pathlib import Path

OUTCOMES=("home","draw","away")
CODES=["1920","2021","2122","2223","2324","2425","2526"]
LABELS={"1920":"2019-20","2021":"2020-21","2122":"2021-22","2223":"2022-23","2324":"2023-24","2425":"2024-25","2526":"2025-26"}

def devig(odds):
    inv={k:1/float(odds[k]) for k in OUTCOMES}; s=sum(inv.values()); return {k:v/s for k,v in inv.items()}

def pois(k,lam): return math.exp(-lam)*lam**k/math.factorial(k)
def scores(hxg,axg,maxg=8,rho=0.0):
    out={}
    for h in range(maxg+1):
      for a in range(maxg+1):
        p=pois(h,hxg)*pois(a,axg)
        if (h,a)==(0,0): p*=max(.05,1-hxg*axg*rho)
        elif (h,a)==(0,1): p*=max(.05,1+hxg*rho)
        elif (h,a)==(1,0): p*=max(.05,1+axg*rho)
        elif (h,a)==(1,1): p*=max(.05,1-rho)
        out[f"{h}-{a}"]=p
    s=sum(out.values()); return {k:v/s for k,v in out.items()}
def wdl(sc):
    r={k:0.0 for k in OUTCOMES}
    for z,p in sc.items():
      h,a=map(int,z.split('-')); r['home' if h>a else 'draw' if h==a else 'away']+=p
    return r
def infer_xg(m,total=2.55):
    strength=math.log(max(m['home'],1e-9)/max(m['away'],1e-9)); share=1/(1+math.exp(-.78*strength))
    h=max(.2,total*share); a=max(.2,total-h); return h,a
def temp_scale(p,t):
    q={k:max(v,1e-15)**(1/t) for k,v in p.items()}; s=sum(q.values()); return {k:v/s for k,v in q.items()}
def actual(r): return {'H':'home','D':'draw','A':'away'}[r['FTR']]
def nll(rows,key): return sum(-math.log(max(r[key][actual(r)],1e-15)) for r in rows)/len(rows)
def brier(rows,key): return sum(sum((r[key][k]-(1 if k==actual(r) else 0))**2 for k in OUTCOMES) for r in rows)/len(rows)
def top1(rows,key): return sum(max(r[key],key=r[key].get)==actual(r) for r in rows)/len(rows)
def ece(rows,key,bins=10):
    bb=[[] for _ in range(bins)]
    for r in rows:
      pred=max(r[key],key=r[key].get); c=r[key][pred]; bb[min(bins-1,int(c*bins))].append((c,1.0 if pred==actual(r) else 0.0))
    return sum(len(x)/len(rows)*abs(sum(a for a,_ in x)/len(x)-sum(b for _,b in x)/len(x)) for x in bb if x)
def metrics(rows,key): return {'samples':len(rows),'nll':nll(rows,key),'brier':brier(rows,key),'ece':ece(rows,key),'top1':top1(rows,key)}
def bootstrap_delta(rows,key_a,key_b,n=3000,seed=475):
    rng=random.Random(seed); losses=[-math.log(max(r[key_a][actual(r)],1e-15))+math.log(max(r[key_b][actual(r)],1e-15)) for r in rows]
    vals=[]
    for _ in range(n): vals.append(sum(rng.choice(losses) for _ in losses)/len(losses))
    vals.sort(); return {'mean':sum(losses)/len(losses),'ci95':[vals[int(.025*(n-1))],vals[int(.975*(n-1))]]}
def roi(rows,key,odds_key,edge):
    bets=[]
    for r in rows:
      for sel in OUTCOMES:
        o=r[odds_key][sel]; ev=r[key][sel]*o-1
        if ev>=edge: bets.append((sel,o,actual(r),ev,r['season']))
    pnl=sum((o-1 if sel==act else -1) for sel,o,act,_,_ in bets)
    return {'bets':len(bets),'roi':pnl/len(bets) if bets else None,'profit_units':pnl,'mean_estimated_edge':sum(x[3] for x in bets)/len(bets) if bets else None}

def load():
    rows=[]; skipped=0
    for code in CODES:
      p=Path('validation-data')/f'E0_{code}.csv'
      with p.open(encoding='utf-8-sig',newline='') as f:
        for r in csv.DictReader(f):
          try:
            if not r.get('FTR') or r['FTR'] not in 'HDA': continue
            op={'home':float(r.get('AvgH') or r['B365H']),'draw':float(r.get('AvgD') or r['B365D']),'away':float(r.get('AvgA') or r['B365A'])}
            cl={'home':float(r.get('AvgCH') or r.get('B365CH') or op['home']),'draw':float(r.get('AvgCD') or r.get('B365CD') or op['draw']),'away':float(r.get('AvgCA') or r.get('B365CA') or op['away'])}
            if min(op.values())<=1 or min(cl.values())<=1: raise ValueError
            m=devig(op); cm=devig(cl); hx,ax=infer_xg(m); sc=scores(hx,ax); base=wdl(sc)
            rows.append({'season':LABELS[code],'FTR':r['FTR'],'FTHG':int(r['FTHG']),'FTAG':int(r['FTAG']),'open_odds':op,'close_odds':cl,'market_open':m,'market_close':cm,'base_model':base,'base_scores':sc})
          except Exception: skipped+=1
    return rows,skipped

def fit_temp(train):
    best=None
    for j in range(50,201):
      t=j/100
      val=sum(-math.log(max(wdl(temp_scale(r['base_scores'],t))[actual(r)],1e-15)) for r in train)/len(train)
      if best is None or val<best[0]: best=(val,t)
    return best[1]

def main():
    rows,skipped=load(); seasons=sorted(set(r['season'] for r in rows)); folds=[]; oos=[]
    for idx in range(3,len(seasons)):
      train=[r for r in rows if r['season'] in seasons[:idx]]; test=[r for r in rows if r['season']==seasons[idx]]; t=fit_temp(train)
      for r in test: r['calibrated']=wdl(temp_scale(r['base_scores'],t))
      oos.extend(test)
      folds.append({'test_season':seasons[idx],'train_seasons':seasons[:idx],'temperature':t,'metrics':{k:metrics(test,k) for k in ['market_open','base_model','calibrated','market_close']}})
    overall={k:metrics(oos,k) for k in ['market_open','base_model','calibrated','market_close']}
    deltas={k:{'nll_vs_open':overall[k]['nll']-overall['market_open']['nll'],'brier_vs_open':overall[k]['brier']-overall['market_open']['brier'],'ece_vs_open':overall[k]['ece']-overall['market_open']['ece']} for k in ['base_model','calibrated','market_close']}
    bootstrap={k:bootstrap_delta(oos,k,'market_open') for k in ['base_model','calibrated','market_close']}
    rois={}
    for key in ['market_open','base_model','calibrated']:
      rois[key]={str(e):roi(oos,key,'open_odds',e) for e in [0.0,.02,.05,.08,.1]}
    result={'schema_version':'v475-real-oos-validation-v1','dataset':{'league':'EPL','seasons':seasons,'rows':len(rows),'oos_seasons':seasons[3:],'oos_rows':len(oos),'skipped':skipped,'source':'football-data.co.uk via direct public CSV'},'protocol':{'split':'strict expanding-window by season','temperature_range':[.5,2.0],'temperature_step':.01,'ai_layer':'not evaluated; no timestamped historical AI facts','odds':'Avg opening and Avg closing 1X2'},'overall_oos':overall,'deltas':deltas,'bootstrap_nll_delta_vs_open':bootstrap,'roi_flat_stake_on_opening_odds':rois,'folds':folds}
    failures=[]
    if deltas['base_model']['nll_vs_open']>=-.002: failures.append('BASE_NLL_NOT_BETTER_THAN_OPEN_MARKET')
    if deltas['calibrated']['nll_vs_open']>=-.002: failures.append('CALIBRATED_NLL_NOT_BETTER_THAN_OPEN_MARKET')
    if bootstrap['calibrated']['ci95'][1]>=0: failures.append('CALIBRATED_NLL_CI_INCLUDES_NO_IMPROVEMENT')
    if (rois['calibrated']['0.05']['bets']<200 or (rois['calibrated']['0.05']['roi'] or -1)<=0): failures.append('CALIBRATED_5PCT_EDGE_ROI_GATE_FAIL')
    result['decision']={'status':'NOT_VALIDATED_FOR_REAL_MONEY' if failures else 'ELIGIBLE_FOR_SHADOW_PROMOTION','failed_gates':failures,'automatic_betting':False}
    Path('validation-output').mkdir(exist_ok=True)
    Path('validation-output/model_effect_validation.json').write_text(json.dumps(result,ensure_ascii=False,indent=2),encoding='utf-8')
    print(json.dumps(result['decision'],ensure_ascii=False))
if __name__=='__main__': main()
