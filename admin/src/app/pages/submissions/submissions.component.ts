import { Component, OnInit, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { API_URL } from '../../core/api-url';

interface SubmissionLog {
  submission_id: number;
  problem_title: string;
  user_email: string;
  language: string;
  verdict: string;
  created_at: string;
}

@Component({
  selector: 'app-submissions',
  template: `
    <h2>Submission Log</h2>

    @if (loading()) {
      <p class="muted">Loading…</p>
    } @else if (submissions().length === 0) {
      <p class="muted">No submissions yet.</p>
    } @else {
      <table class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Problem</th>
            <th>User</th>
            <th>Language</th>
            <th>Verdict</th>
            <th>Submitted</th>
          </tr>
        </thead>
        <tbody>
          @for (s of submissions(); track s.submission_id) {
            <tr>
              <td>{{ s.submission_id }}</td>
              <td>{{ s.problem_title }}</td>
              <td>{{ s.user_email }}</td>
              <td>{{ s.language }}</td>
              <td><span class="verdict verdict-{{ s.verdict }}">{{ s.verdict }}</span></td>
              <td>{{ formatDate(s.created_at) }}</td>
            </tr>
          }
        </tbody>
      </table>
    }

    @if (error()) { <p class="error">{{ error() }}</p> }
  `,
  styleUrl: './submissions.component.css',
})
export class SubmissionsComponent implements OnInit {
  private http = inject(HttpClient);
  submissions = signal<SubmissionLog[]>([]);
  loading = signal(true);
  error = signal('');

  ngOnInit(): void { this.load(); }

  private async load(): Promise<void> {
    try {
      const data = await firstValueFrom(this.http.get<SubmissionLog[]>(`${API_URL}/admin/submissions`));
      this.submissions.set(data ?? []);
    } catch {
      this.error.set('Failed to load submissions.');
    } finally {
      this.loading.set(false);
    }
  }

  formatDate(iso: string): string {
    return new Date(iso).toLocaleString();
  }
}
