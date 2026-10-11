from mb import MB
m=MB('emby');m.login()
c,r,_=m.call('GET','/Items',Recursive='true',IncludeItemTypes='Movie,Episode',Fields='Path,MediaSources,PresentationUniqueKey,MediaStreams')
for i in r['Items']:
    print(i['Type'],i['Name'],i['Id'],i.get('Path'),'PUK',i.get('PresentationUniqueKey'))
    for s in i.get('MediaSources',[]):
        subs=[(x.get('Index'),x.get('Language'),x.get('IsExternal'),x.get('Path')) for x in s.get('MediaStreams',[]) if x['Type']=='Subtitle']
        print('    MS',s['Id'],repr(s.get('Name')),'subs',subs)
