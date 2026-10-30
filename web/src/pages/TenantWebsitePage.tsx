import { useEffect, useMemo, useState } from 'react';
import { Alert, Box, Button, Card, CardContent, CircularProgress, Divider, FormControlLabel, MenuItem, Stack, Switch, TextField, Typography } from '@mui/material';
import Grid from '@mui/material/GridLegacy';
import { useNotify } from 'react-admin';
import { SaveOutlined as SaveIcon, PublishOutlined as PublishIcon, UndoOutlined as UndoIcon } from '@mui/icons-material';
import { apiRequest } from '../utils/apiClient';
import { PageHeader } from '../components/Enterprise';

type WebsiteSection = { key: string; title: string; subtitle: string; body: string; image_url: string; cta_label: string; cta_url: string; visible: boolean; order: number };
type WebsiteContent = { sections: WebsiteSection[] };
type WebsiteVersion = { version: number; created_at: string };
const sectionKeys = ['header','hero','announcement','about','services','coverage','sla','contact','social','footer'];
const defaultTitles: Record<string,string> = { header:'Header',hero:'Hero',announcement:'Announcement',about:'About us',services:'Services',coverage:'Coverage',sla:'Service level',contact:'Contact',social:'Social media',footer:'Footer' };
const emptyContent: WebsiteContent = { sections: sectionKeys.map((key,index) => ({ key, title: defaultTitles[key], subtitle:'', body:'', image_url:'', cta_label:'', cta_url:'', visible:true, order:index })) };

const TenantWebsitePage = () => {
  const notify = useNotify();
  const [content, setContent] = useState<WebsiteContent>(emptyContent);
  const [versions, setVersions] = useState<WebsiteVersion[]>([]);
  const [selectedKey, setSelectedKey] = useState('hero');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);
  const [error, setError] = useState('');
  const selected = useMemo(() => content.sections.find(section => section.key === selectedKey) || content.sections[0], [content, selectedKey]);

  const refreshVersions = async () => {
    const rows = await apiRequest<WebsiteVersion[]>('/tenant-website/versions');
    setVersions(rows || []);
  };
  useEffect(() => {
    let active = true;
    Promise.all([apiRequest<{draft:WebsiteContent}>('/tenant-website/draft'),apiRequest<WebsiteVersion[]>('/tenant-website/versions')])
      .then(([draft, history]) => { if (active) { setContent({ ...emptyContent, ...(draft?.draft || {}) }); setVersions(history || []); } })
      .catch((err: Error) => { if (active) setError(err.message || 'Unable to load tenant website'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, []);

  const updateSection = (patch: Partial<WebsiteSection>) => {
    setContent(current => ({ ...current, sections: current.sections.map(section => section.key === selectedKey ? { ...section, ...patch } : section) }));
  };
  const saveDraft = async () => {
    setSaving(true); setError('');
    try {
      const result = await apiRequest<{draft:WebsiteContent}>('/tenant-website/draft', { method:'PUT', body:JSON.stringify(content) });
      setContent({ ...emptyContent, ...(result?.draft || content) });
      notify('Website draft saved. It is not public until you publish.',{type:'success'});
    } catch (err) { const message = err instanceof Error ? err.message : 'Save failed'; setError(message); notify(message,{type:'error'}); }
    finally { setSaving(false); }
  };
  const publish = async () => {
    setPublishing(true); setError('');
    try { await saveDraft(); await apiRequest('/tenant-website/publish',{method:'POST'}); await refreshVersions(); notify('Tenant website published.',{type:'success'}); }
    catch (err) { const message = err instanceof Error ? err.message : 'Publish failed'; setError(message); notify(message,{type:'error'}); }
    finally { setPublishing(false); }
  };
  const rollback = async (version: number) => {
    setRollingBack(true); setError('');
    try { const result = await apiRequest<{draft:WebsiteContent}>('/tenant-website/versions/'+version+'/rollback',{method:'POST'}); setContent({ ...emptyContent, ...(result?.draft || content) }); await refreshVersions(); notify('Selected version restored and published as a new version.',{type:'success'}); }
    catch (err) { const message = err instanceof Error ? err.message : 'Rollback failed'; setError(message); notify(message,{type:'error'}); }
    finally { setRollingBack(false); }
  };

  if (loading) return <Box display="flex" justifyContent="center" minHeight="60vh" alignItems="center"><CircularProgress /></Box>;
  if (!selected) return <Alert severity="warning">No website sections are available.</Alert>;

  return (
    <Box sx={{p:{xs:2,md:3},maxWidth:1450,mx:'auto'}}>
      <PageHeader section="TENANT EXPERIENCE" title="WEBSITE STUDIO" subtitle="Edit structured website sections, save drafts, publish and restore previous versions. Content is rendered as text, never executable HTML." />
      {error && <Alert severity="error" sx={{mb:2}}>{error}</Alert>}
      <Alert severity="info" sx={{mb:2}}>Saving a draft does not change the public website. Publish creates an immutable version; rollback republishes an earlier snapshot.</Alert>
      <Grid container spacing={2}>
        <Grid item xs={12} lg={7}>
          <Card variant="outlined"><CardContent>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{mb:2}}>
              <Typography variant="h6" fontWeight={800}>Section editor</Typography>
              <TextField select size="small" label="Section" value={selectedKey} onChange={event=>setSelectedKey(event.target.value)} sx={{minWidth:210}}>
                {sectionKeys.map(key=><MenuItem value={key} key={key}>{defaultTitles[key]}</MenuItem>)}
              </TextField>
            </Stack>
            <Divider sx={{mb:2}} />
            <FormControlLabel control={<Switch checked={selected.visible} onChange={event=>updateSection({visible:event.target.checked})}/>} label="Show this section on the public website" />
            <Grid container spacing={2} sx={{mt:1}}>
              <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="Section title" value={selected.title} onChange={event=>updateSection({title:event.target.value})}/></Grid>
              <Grid item xs={12} sm={6}><TextField fullWidth size="small" type="number" label="Display order" value={selected.order} onChange={event=>updateSection({order:Math.max(0,Number(event.target.value)||0)})}/></Grid>
              <Grid item xs={12}><TextField fullWidth size="small" label="Subtitle" value={selected.subtitle} onChange={event=>updateSection({subtitle:event.target.value})}/></Grid>
              <Grid item xs={12}><TextField fullWidth size="small" multiline minRows={4} label="Body text" value={selected.body} onChange={event=>updateSection({body:event.target.value})} helperText="Plain text only. HTML and scripts are not rendered."/></Grid>
              <Grid item xs={12}><TextField fullWidth size="small" label="Image URL" value={selected.image_url} onChange={event=>updateSection({image_url:event.target.value})} helperText="Same-origin relative path or HTTPS URL."/></Grid>
              <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="CTA label" value={selected.cta_label} onChange={event=>updateSection({cta_label:event.target.value})}/></Grid>
              <Grid item xs={12} sm={6}><TextField fullWidth size="small" label="CTA URL" value={selected.cta_url} onChange={event=>updateSection({cta_url:event.target.value})} helperText="HTTPS, mailto, tel or same-origin relative path."/></Grid>
            </Grid>
          </CardContent></Card>
          <Stack direction="row" justifyContent="flex-end" spacing={1} sx={{mt:2}}>
            <Button variant="outlined" onClick={()=>void saveDraft()} disabled={saving||publishing} startIcon={<SaveIcon/>}>{saving?'Saving…':'Save draft'}</Button>
            <Button variant="contained" onClick={()=>void publish()} disabled={saving||publishing} startIcon={publishing?<CircularProgress size={16}/>:<PublishIcon/>}>{publishing?'Publishing…':'Publish website'}</Button>
          </Stack>
        </Grid>
        <Grid item xs={12} lg={5}>
          <Card variant="outlined"><CardContent>
            <Typography variant="h6" fontWeight={800}>Live draft preview</Typography>
            <Divider sx={{my:2}}/>
            <Box sx={{p:3,bgcolor:'#101827',color:'#fff',borderRadius:1,border:'1px solid #263449'}}>
              <Typography variant="overline" color="primary">{selected.key.toUpperCase()} · {selected.visible?'VISIBLE':'HIDDEN'}</Typography>
              <Typography variant="h4" fontWeight={900} sx={{my:1}}>{selected.title || defaultTitles[selected.key]}</Typography>
              {selected.subtitle && <Typography variant="h6" color="#CBD5E1">{selected.subtitle}</Typography>}
              {selected.image_url && <Box component="img" src={selected.image_url} alt={selected.title} sx={{width:'100%',maxHeight:180,objectFit:'cover',my:2}}/>}
              <Typography sx={{whiteSpace:'pre-wrap',mt:2,color:'#DCE5F4'}}>{selected.body}</Typography>
              {selected.cta_label && <Button disabled variant="contained" sx={{mt:2}}>{selected.cta_label}</Button>}
            </Box>
          </CardContent></Card>
          <Card variant="outlined" sx={{mt:2}}><CardContent>
            <Stack direction="row" justifyContent="space-between" alignItems="center"><Typography variant="h6" fontWeight={800}>Published versions</Typography><Button size="small" onClick={()=>void refreshVersions()}>Refresh</Button></Stack>
            <Divider sx={{my:2}}/>
            {!versions.length ? <Typography color="text.secondary" variant="body2">No published versions yet.</Typography> : versions.map(version=><Stack key={version.version} direction="row" justifyContent="space-between" alignItems="center" sx={{py:1,borderBottom:'1px solid',borderColor:'divider'}}><Box><Typography fontWeight={700}>Version {version.version}</Typography><Typography variant="caption" color="text.secondary">{new Date(version.created_at).toLocaleString()}</Typography></Box><Button size="small" color="warning" startIcon={<UndoIcon/>} disabled={rollingBack} onClick={()=>void rollback(version.version)}>Restore</Button></Stack>)}
          </CardContent></Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default TenantWebsitePage;