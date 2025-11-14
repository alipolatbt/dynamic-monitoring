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
  ThemeProvider,
  createTheme,
  Stepper,
  Step,
  StepLabel,
  useMediaQuery,
  Alert,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon, PlayArrow as PlayIcon, Settings as SettingsIcon } from '@mui/icons-material';
import axios from 'axios';

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1976d2' },
    background: { default: '#f5f5f5', paper: '#fff' },
  },
  typography: {
    fontSize: 14,
    h6: { fontSize: '1.1rem' },
    body1: { fontSize: '0.9rem' },
    body2: { fontSize: '0.8rem' },
  },
  components: {
    MuiTableCell: {
      styleOverrides: {
        root: { padding: '8px 16px' },
      },
    },
  },
});

type TokenRequest = { url: string; method: string; headers: Record<string,string>; body?: any };
type RequestConfig = { url: string; method: string; headers?: Record<string,string>; body?: any; token_header?: string };
type APICheck = { name: string; token_request: TokenRequest; token_path: string; requests: RequestConfig[]; interval: number; timeout: number };
type CheckResult = { name: string; url: string; status_code: number; duration: number; timestamp: string; error?: string };

type SMTPSettings = { host: string; port: number; username: string; password: string; from: string; to: string; enabled: boolean };

const INITIAL_CHECK: APICheck = {
  name: 'Related Digital API Flow',
  token_request: {
    url: 'https://api.relateddigital.com/resta/api/auth/login',
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: {
      UserName: 'emreust',
      Password: '9DE57C32'
    }
  },
  token_path: 'ServiceTicket',
  requests: [{
    url: 'http://api.relateddigital.com/resta/api/post/PostHtml',
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    token_header: 'Authorization',
    body: {
      FromName: 'Emre Üst -LIVE API',
      FromAddress: 'emre.ust@euromsg.net',
      ReplyAddress: 'noreply@emreust.com',
      Subject: 'LIVE API MAIL',
      HtmlBody: '<html><head></head><body>Hello World!!</body></html>',
      Charset: 'iso-8859-9',
      ToName: 'Emre Üst',
      ToEmailAddress: 'emre.ust@euromsg.com',
      PostType: '',
      KeyId: '',
      CustomParams: ''
    }
  }],
  interval: 300,
  timeout: 30
};

export default function App() {
  const [checks, setChecks] = useState<APICheck[]>([]);
  const [results, setResults] = useState<CheckResult[]>([]);
  const [open, setOpen] = useState(false);
  const [smtpOpen, setSmtpOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeStep, setActiveStep] = useState(0);
  const [newCheck, setNewCheck] = useState<APICheck>(INITIAL_CHECK);
  const [smtpSettings, setSMTPSettings] = useState<SMTPSettings>({
    host: '',
    port: 587,
    username: '',
    password: '',
    from: '',
    to: '',
    enabled: false
  });
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  useEffect(() => {
    fetchChecks();
    fetchResults();
    fetchSMTP();
    const id = setInterval(fetchResults, 5000);
    return () => clearInterval(id);
  }, []);

  async function fetchChecks() {
    try {
      const r = await axios.get('/api/checks');
      setChecks(Object.values(r.data));
    } catch (e) {
      setError('Failed to fetch checks');
    }
  }

  async function fetchResults() {
    try {
      const r = await axios.get('/api/results');
      setResults(r.data);
    } catch (e) {
      setError('Failed to fetch results');
    }
  }

  async function fetchSMTP() {
    try {
      const r = await axios.get('/api/smtp');
      setSMTPSettings(r.data);
    } catch (e) {
      // Ignore error on first load
    }
  }

  async function saveSMTP() {
    try {
      setLoading(true);
      await axios.post('/api/smtp', smtpSettings);
      setSmtpOpen(false);
      setError(null);
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to save SMTP settings');
    } finally {
      setLoading(false);
    }
  }

  async function handleSubmit() {
    try {
      setLoading(true);
      setError(null);
      await axios.post('/api/checks', newCheck);
      setOpen(false);
      setNewCheck(INITIAL_CHECK);
      fetchChecks();
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to create check');
    } finally {
      setLoading(false);
    }
  }

  async function handleTrigger(name: string) {
    try {
      await axios.post(`/api/checks/${encodeURIComponent(name)}/trigger`);
      setTimeout(fetchResults, 1000);
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to trigger check');
    }
  }

  async function handleDelete(name: string) {
    if (!window.confirm(`Are you sure you want to delete "${name}"?`)) return;
    try {
      await axios.delete(`/api/checks/${encodeURIComponent(name)}`);
      fetchChecks();
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to delete check');
    }
  }

  const handleNext = () => setActiveStep(s => s + 1);
  const handleBack = () => setActiveStep(s => s - 1);

  const stats = {
    totalChecks: checks.length,
    activeEndpoints: checks.reduce((a: number, c) => a + (c.requests?.length || 0), 0),
    avgResponseTime: results.length ? (results.reduce((a: number, r) => a + r.duration, 0) / results.length).toFixed(2) : '0',
    errorRate: results.length ? ((results.filter(r => r.error || r.status_code >= 400).length / results.length) * 100).toFixed(1) : '0'
  };

  const renderStepContent = (step: number) => {
    switch (step) {
      case 0:
        return (
          <Box sx={{ mt: 2 }}>
            <TextField fullWidth label="Monitor Name" sx={{ mb: 2 }} value={newCheck.name} onChange={e => setNewCheck({ ...newCheck, name: e.target.value })} />
            <Typography variant="subtitle2" sx={{ mb: 1 }}>Step 1: Authentication Details</Typography>
            <TextField fullWidth label="Auth URL" sx={{ mb: 2 }} value={newCheck.token_request.url} onChange={e => setNewCheck({ ...newCheck, token_request: { ...newCheck.token_request, url: e.target.value } })} />
            <TextField fullWidth label="Token Path (e.g. ServiceTicket)" sx={{ mb: 2 }} value={newCheck.token_path} onChange={e => setNewCheck({ ...newCheck, token_path: e.target.value })} />
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField fullWidth label="Username" value={newCheck.token_request.body?.UserName} onChange={e => setNewCheck({ ...newCheck, token_request: { ...newCheck.token_request, body: { ...newCheck.token_request.body, UserName: e.target.value } } })} />
              </Grid>
              <Grid item xs={6}>
                <TextField fullWidth label="Password" type="password" value={newCheck.token_request.body?.Password} onChange={e => setNewCheck({ ...newCheck, token_request: { ...newCheck.token_request, body: { ...newCheck.token_request.body, Password: e.target.value } } })} />
              </Grid>
            </Grid>
          </Box>
        );
      case 1:
        return (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>Step 2: PostHtml API Details</Typography>
            <TextField fullWidth label="API URL" sx={{ mb: 2 }} value={newCheck.requests[0].url} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], url: e.target.value }] })} />
            <Grid container spacing={2}>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="From Name" sx={{ mb: 2 }} value={newCheck.requests[0].body?.FromName} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], body: { ...newCheck.requests[0].body, FromName: e.target.value } }] })} />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="From Email" sx={{ mb: 2 }} value={newCheck.requests[0].body?.FromAddress} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], body: { ...newCheck.requests[0].body, FromAddress: e.target.value } }] })} />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="To Name" sx={{ mb: 2 }} value={newCheck.requests[0].body?.ToName} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], body: { ...newCheck.requests[0].body, ToName: e.target.value } }] })} />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="To Email" sx={{ mb: 2 }} value={newCheck.requests[0].body?.ToEmailAddress} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], body: { ...newCheck.requests[0].body, ToEmailAddress: e.target.value } }] })} />
              </Grid>
              <Grid item xs={12}>
                <TextField fullWidth label="Subject" sx={{ mb: 2 }} value={newCheck.requests[0].body?.Subject} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], body: { ...newCheck.requests[0].body, Subject: e.target.value } }] })} />
              </Grid>
              <Grid item xs={12}>
                <TextField fullWidth multiline rows={4} label="HTML Body" sx={{ mb: 2 }} value={newCheck.requests[0].body?.HtmlBody} onChange={e => setNewCheck({ ...newCheck, requests: [{ ...newCheck.requests[0], body: { ...newCheck.requests[0].body, HtmlBody: e.target.value } }] })} />
              </Grid>
            </Grid>
          </Box>
        );
      case 2:
        return (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" sx={{ mb: 1 }}>Step 3: Monitoring Settings</Typography>
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField fullWidth type="number" label="Check Interval (seconds)" value={newCheck.interval} onChange={e => setNewCheck({ ...newCheck, interval: parseInt(e.target.value) || 300 })} />
              </Grid>
              <Grid item xs={6}>
                <TextField fullWidth type="number" label="Timeout (seconds)" value={newCheck.timeout} onChange={e => setNewCheck({ ...newCheck, timeout: parseInt(e.target.value) || 30 })} />
              </Grid>
            </Grid>
          </Box>
        );
      default:
        return null;
    }
  };

  return (
    <ThemeProvider theme={theme}>
      <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
        <AppBar position="static" sx={{ bgcolor: '#1976d2' }}>
          <Toolbar sx={{ display: 'flex', gap: 2 }}>
            <img src="https://www.dogustech.com/img/logo.svg" alt="Doğuş Teknoloji" style={{ width: 120, height: 40 }} onError={(e) => { e.currentTarget.style.display = 'none' }} />
            <Typography variant="h6" sx={{ fontSize: '1.2rem', color: 'white' }}>Doğuş Teknoloji API Monitoring</Typography>
            <Box sx={{ flexGrow: 1 }} />
            <Button variant="outlined" size="small" sx={{ color: 'white', borderColor: 'white', mr: 1 }} startIcon={<SettingsIcon />} onClick={() => setSmtpOpen(true)}>
              SMTP Settings
            </Button>
            <Button variant="contained" color="primary" size="small" sx={{ bgcolor: '#2196f3' }} startIcon={<AddIcon />} onClick={() => setOpen(true)}>
              Add Check
            </Button>
          </Toolbar>
        </AppBar>

        <Container maxWidth="xl" sx={{ mt: 3, mb: 3 }}>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2">Total Checks</Typography><Typography variant="h6">{stats.totalChecks}</Typography></CardContent></Card></Grid>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2">Active Endpoints</Typography><Typography variant="h6">{stats.activeEndpoints}</Typography></CardContent></Card></Grid>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2">Avg Response Time</Typography><Typography variant="h6">{stats.avgResponseTime}s</Typography></CardContent></Card></Grid>
            <Grid item xs={12} sm={6} md={3}><Card><CardContent><Typography variant="body2">Error Rate</Typography><Typography variant="h6">{stats.errorRate}%</Typography></CardContent></Card></Grid>
          </Grid>

          <Paper sx={{ mt: 2, p: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>Active Checks</Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Name</TableCell>
                    <TableCell>Auth URL</TableCell>
                    <TableCell>API URL</TableCell>
                    <TableCell>Interval</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {checks.map(c => (
                    <TableRow key={c.name}>
                      <TableCell>{c.name}</TableCell>
                      <TableCell>{c.token_request.url}</TableCell>
                      <TableCell>{c.requests[0].url}</TableCell>
                      <TableCell>{c.interval}s</TableCell>
                      <TableCell>
                        <IconButton size="small" color="primary" onClick={() => handleTrigger(c.name)} title="Run now"><PlayIcon fontSize="small" /></IconButton>
                        <IconButton size="small" color="error" onClick={() => handleDelete(c.name)} title="Delete"><DeleteIcon fontSize="small" /></IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>

          <Paper sx={{ mt: 2, p: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>Recent Results</Typography>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Check</TableCell>
                    <TableCell>URL</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Duration</TableCell>
                    <TableCell>Time</TableCell>
                    <TableCell>Error</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.map((r, i) => (
                    <TableRow key={i} sx={{ bgcolor: r.error || r.status_code >= 400 ? 'rgba(255,23,68,0.08)' : undefined }}>
                      <TableCell>{r.name}</TableCell>
                      <TableCell>{r.url}</TableCell>
                      <TableCell><Chip size="small" label={r.status_code || 'ERR'} color={r.error || r.status_code >= 400 ? 'error' : 'success'} /></TableCell>
                      <TableCell><Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>{r.duration ? r.duration.toFixed(3) + 's' : '-'}<LinearProgress variant="determinate" value={Math.min((r.duration / 2) * 100, 100)} sx={{ width: 60 }} /></Box></TableCell>
                      <TableCell>{new Date(r.timestamp).toLocaleString()}</TableCell>
                      <TableCell><Typography variant="caption" color="error">{r.error || '-'}</Typography></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        </Container>

        <Dialog open={open} onClose={() => setOpen(false)} maxWidth="md" fullWidth>
          <DialogTitle>New API Check</DialogTitle>
          <DialogContent>
            <Stepper activeStep={activeStep} sx={{ mt: 2 }}>
              <Step><StepLabel>Authentication</StepLabel></Step>
              <Step><StepLabel>API Details</StepLabel></Step>
              <Step><StepLabel>Monitor Settings</StepLabel></Step>
            </Stepper>
            {renderStepContent(activeStep)}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setOpen(false)}>Cancel</Button>
            <Box sx={{ flexGrow: 1 }} />
            {activeStep > 0 && <Button onClick={handleBack}>Back</Button>}
            {activeStep < 2 ? (
              <Button variant="contained" onClick={handleNext}>Next</Button>
            ) : (
              <Button variant="contained" onClick={handleSubmit} disabled={loading}>
                {loading ? <CircularProgress size={20} /> : 'Create Check'}
              </Button>
            )}
          </DialogActions>
        </Dialog>

        <Dialog open={smtpOpen} onClose={() => setSmtpOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>SMTP Email Alert Settings</DialogTitle>
          <DialogContent>
            <Box sx={{ mt: 2 }}>
              <TextField fullWidth label="SMTP Host" sx={{ mb: 2 }} value={smtpSettings.host} onChange={e => setSMTPSettings({ ...smtpSettings, host: e.target.value })} placeholder="smtp.gmail.com" />
              <TextField fullWidth type="number" label="SMTP Port" sx={{ mb: 2 }} value={smtpSettings.port} onChange={e => setSMTPSettings({ ...smtpSettings, port: parseInt(e.target.value) || 587 })} />
              <TextField fullWidth label="Username" sx={{ mb: 2 }} value={smtpSettings.username} onChange={e => setSMTPSettings({ ...smtpSettings, username: e.target.value })} />
              <TextField fullWidth type="password" label="Password" sx={{ mb: 2 }} value={smtpSettings.password} onChange={e => setSMTPSettings({ ...smtpSettings, password: e.target.value })} />
              <TextField fullWidth label="From Email" sx={{ mb: 2 }} value={smtpSettings.from} onChange={e => setSMTPSettings({ ...smtpSettings, from: e.target.value })} placeholder="alerts@example.com" />
              <TextField fullWidth label="To Email" sx={{ mb: 2 }} value={smtpSettings.to} onChange={e => setSMTPSettings({ ...smtpSettings, to: e.target.value })} placeholder="admin@example.com" />
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <input type="checkbox" checked={smtpSettings.enabled} onChange={e => setSMTPSettings({ ...smtpSettings, enabled: e.target.checked })} />
                <Typography variant="body2">Enable email alerts</Typography>
              </Box>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setSmtpOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={saveSMTP} disabled={loading}>
              {loading ? <CircularProgress size={20} /> : 'Save Settings'}
            </Button>
          </DialogActions>
        </Dialog>
      </Box>
    </ThemeProvider>
  );
}