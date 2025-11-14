import React, { useEffect, useState } from 'react';
import {
  AppBar,
  Toolbar,
  Box,
  Container,
  Typography,
  Button,
  Grid,
  Card,
  CardContent,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  IconButton,
  CircularProgress,
  LinearProgress,
  createTheme,
  ThemeProvider,
  useMediaQuery,
  useTheme,
  Alert,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon, Speed as SpeedIcon, Error as ErrorIcon, CheckCircle as CheckCircleIcon } from '@mui/icons-material';
import axios from 'axios';

const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#00b0ff' },
    background: { default: '#0a1929', paper: '#132f4c' },
  },
});

type TokenRequest = { url: string; method: string; headers: Record<string,string>; body?: any };
type RequestConfig = { url: string; method: string; headers?: Record<string,string>; body?: any; token_header?: string };
type APICheck = { name: string; token_request: TokenRequest; token_path: string; requests: RequestConfig[]; interval: number; timeout: number };
type CheckResult = { name: string; url: string; status_code: number; duration: number; timestamp: string; error?: string };

export default function App(){
  const [checks, setChecks] = useState<APICheck[]>([]);
  const [results, setResults] = useState<CheckResult[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string| null>(null);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  const [newCheck, setNewCheck] = useState<APICheck>({
    name: '',
    token_request: { url: '', method: 'POST', headers: { 'Content-Type': 'application/json' } },
    token_path: 'token',
    requests: [{ url: '', method: 'GET', headers: { 'Content-Type': 'application/json' }, token_header: 'Authorization' }],
    interval: 60,
    timeout: 10
  });

  useEffect(()=>{ fetchChecks(); fetchResults(); const id = setInterval(fetchResults,5000); return ()=>clearInterval(id) },[]);

  async function fetchChecks(){
    try{ const r = await axios.get('/api/checks'); setChecks(Object.values(r.data)); }catch(e){ setError('Failed to fetch checks') }
  }
  async function fetchResults(){
    try{ const r = await axios.get('/api/results'); setResults(r.data); }catch(e){ setError('Failed to fetch results') }
  }
  async function handleSubmit(){
    try{ setLoading(true); await axios.post('/api/checks', newCheck); setOpen(false); fetchChecks(); }
    catch(e){ setError('Failed to create check') }
    finally{ setLoading(false) }
  }

  const stats = { totalChecks: checks.length, activeEndpoints: checks.reduce((a,c)=>a + (c.requests?.length||0),0), avgResponseTime: results.length? (results.reduce((a,r)=>a+r.duration,0)/results.length).toFixed(2):'0', errorRate: results.length? ((results.filter(r=>r.error||r.status_code>=400).length/results.length)*100).toFixed(1):'0' };

  return (
    <ThemeProvider theme={darkTheme}>
      <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
        <AppBar position="static" color="transparent" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Toolbar sx={{ display:'flex', gap:2 }}>
            <img src="/logo.svg" alt="PLT" style={{ width:48, height:48 }} />
            <Box>
              <Typography variant="h6">PLT Monitor</Typography>
              <Typography variant="caption" color="primary">Performance & Load Testing</Typography>
            </Box>
            <Box sx={{ flexGrow:1 }} />
            <Button variant="contained" startIcon={<AddIcon />} onClick={()=>setOpen(true)}>Add New Check</Button>
          </Toolbar>
        </AppBar>
        <Container maxWidth="xl" sx={{ mt:4, mb:4 }}>
          {error && <Alert severity="error" sx={{ mb:2 }}>{error}</Alert>}
          <Grid container spacing={3} sx={{ mb:3 }}>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography>Total Checks</Typography><Typography variant="h4">{stats.totalChecks}</Typography></CardContent></Card></Grid>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography>Active Endpoints</Typography><Typography variant="h4">{stats.activeEndpoints}</Typography></CardContent></Card></Grid>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography>Avg Response Time</Typography><Typography variant="h4">{stats.avgResponseTime}s</Typography></CardContent></Card></Grid>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography>Error Rate</Typography><Typography variant="h4">{stats.errorRate}%</Typography></CardContent></Card></Grid>
          </Grid>

          <Grid container spacing={3}>
            <Grid item xs={12}><Paper sx={{ p:3 }}>
              <Typography variant="h6" sx={{ mb:2 }}>Active Checks</Typography>
              <TableContainer>
                <Table>
                  <TableHead><TableRow><TableCell>Name</TableCell><TableCell>Token URL</TableCell><TableCell>Monitored</TableCell><TableCell>Interval</TableCell><TableCell>Actions</TableCell></TableRow></TableHead>
                  <TableBody>{checks.map(c=> (
                    <TableRow key={c.name}><TableCell>{c.name}</TableCell><TableCell>{c.token_request.url}</TableCell><TableCell>{(c.requests||[]).map(r=>r.url).join(', ')}</TableCell><TableCell>{c.interval}</TableCell><TableCell><IconButton color="error"><DeleteIcon/></IconButton></TableCell></TableRow>
                  ))}</TableBody>
                </Table>
              </TableContainer>
            </Paper></Grid>

            <Grid item xs={12}><Paper sx={{ p:3 }}>
              <Typography variant="h6" sx={{ mb:2 }}>Recent Results</Typography>
              <TableContainer>
                <Table>
                  <TableHead><TableRow><TableCell>Name</TableCell><TableCell>URL</TableCell><TableCell>Status</TableCell><TableCell>Duration(s)</TableCell><TableCell>Time</TableCell><TableCell>Error</TableCell></TableRow></TableHead>
                  <TableBody>{results.map((r,i)=> (
                    <TableRow key={i} sx={{ bgcolor: r.error || r.status_code>=400 ? 'rgba(255,23,68,0.08)': undefined }}>
                      <TableCell>{r.name}</TableCell>
                      <TableCell>{r.url}</TableCell>
                      <TableCell><Chip label={r.status_code||'ERR'} color={r.error||r.status_code>=400? 'error':'success'} size="small"/></TableCell>
                      <TableCell><Box sx={{ display:'flex', alignItems:'center', gap:1 }}>{r.duration.toFixed(3)}<LinearProgress variant="determinate" value={Math.min((r.duration/2)*100,100)} sx={{ width:80 }} /></Box></TableCell>
                      <TableCell>{new Date(r.timestamp).toLocaleString()}</TableCell>
                      <TableCell>{r.error||'-'}</TableCell>
                    </TableRow>
                  ))}</TableBody>
                </Table>
              </TableContainer>
            </Paper></Grid>
          </Grid>
        </Container>

        <Dialog open={open} onClose={()=>setOpen(false)} fullWidth maxWidth="md">
          <DialogTitle>Add New API Check</DialogTitle>
          <DialogContent>
            <TextField fullWidth label="Name" sx={{ mt:1 }} value={newCheck.name} onChange={e=>setNewCheck({...newCheck, name:e.target.value})} />
            <TextField fullWidth label="Token URL" sx={{ mt:1 }} value={newCheck.token_request.url} onChange={e=>setNewCheck({...newCheck, token_request:{...newCheck.token_request, url:e.target.value}})} />
            <TextField fullWidth label="Token Path (e.g. data.access_token)" sx={{ mt:1 }} value={newCheck.token_path} onChange={e=>setNewCheck({...newCheck, token_path:e.target.value})} />
            <TextField fullWidth label="API URL" sx={{ mt:1 }} value={newCheck.requests[0].url} onChange={e=>setNewCheck({...newCheck, requests:[{...newCheck.requests[0], url:e.target.value}]})} />
            <Box sx={{ display:'flex', gap:2, mt:2 }}>
              <TextField label="Interval (s)" type="number" value={newCheck.interval} onChange={e=>setNewCheck({...newCheck, interval: parseInt(e.target.value||'0')})} />
              <TextField label="Timeout (s)" type="number" value={newCheck.timeout} onChange={e=>setNewCheck({...newCheck, timeout: parseInt(e.target.value||'0')})} />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={()=>setOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={handleSubmit} disabled={loading}>{loading? <CircularProgress size={20}/>:'Add Check'}</Button>
          </DialogActions>
        </Dialog>
      </Box>
    </ThemeProvider>
  )
}
