import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Alert, Box, Button, CircularProgress, Container, Stack, Typography } from '@mui/material';
import { apiRequest } from '../utils/apiClient';

type WebsiteSection = { key:string; title:string; subtitle:string; body:string; image_url:string; cta_label:string; cta_url:string; visible:boolean; order:number };
type WebsiteContent = { sections:WebsiteSection[] };

const TenantPublicWebsitePage = () => {
  const { identifier = '' } = useParams();
  const [content,setContent] = useState<WebsiteContent|null>(null);
  const [loading,setLoading] = useState(true);
  const [error,setError] = useState('');
  useEffect(()=>{
    let active=true;
    apiRequest<WebsiteContent>('/tenant-website/public/'+encodeURIComponent(identifier))
      .then(value=>{if(active)setContent(value);})
      .catch((err:Error)=>{if(active)setError(err.message||'Website is not published');})
      .finally(()=>{if(active)setLoading(false);});
    return ()=>{active=false;};
  },[identifier]);
  if(loading)return <Box display="flex" justifyContent="center" alignItems="center" minHeight="80vh"><CircularProgress/></Box>;
  if(error||!content)return <Container maxWidth="md" sx={{py:8}}><Alert severity="info">{error||'This tenant website is not published.'}</Alert></Container>;
  const sections=[...content.sections].filter(section=>section.visible).sort((a,b)=>a.order-b.order);
  return <Box sx={{minHeight:'100vh',bgcolor:'#0B1220',color:'#F8FAFC',py:{xs:3,md:6}}}><Container maxWidth="lg">
    {sections.map(section=><Box key={section.key} component="section" sx={{py:{xs:3,md:5},borderBottom:'1px solid rgba(255,255,255,0.08)'}}>
      <Typography variant="overline" color="primary">{section.key.toUpperCase()}</Typography>
      <Typography variant="h3" fontWeight={900} sx={{my:1,fontSize:{xs:'2rem',md:'3rem'}}}>{section.title}</Typography>
      {section.subtitle&&<Typography variant="h6" color="#CBD5E1" sx={{mb:2}}>{section.subtitle}</Typography>}
      {section.image_url&&<Box component="img" src={section.image_url} alt={section.title} sx={{width:'100%',maxHeight:420,objectFit:'cover',borderRadius:2,my:2}}/>}
      {section.body&&<Typography sx={{whiteSpace:'pre-wrap',maxWidth:900,lineHeight:1.8,color:'#D8E1EF'}}>{section.body}</Typography>}
      {section.cta_label&&section.cta_url&&<Stack direction="row" sx={{mt:3}}><Button variant="contained" href={section.cta_url} target={section.cta_url.startsWith('https://')?'_blank':undefined} rel={section.cta_url.startsWith('https://')?'noopener noreferrer':undefined}>{section.cta_label}</Button></Stack>}
    </Box>)}
  </Container></Box>;
};

export default TenantPublicWebsitePage;