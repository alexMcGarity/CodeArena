import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../core/auth.service';

@Component({
  selector: 'app-login',
  imports: [ReactiveFormsModule],
  template: `
    <div class="login-page">
      <div class="login-card">
        <h2>CodeArena Admin</h2>
        <form [formGroup]="form" (ngSubmit)="submit()">
          <label>
            Email
            <input type="email" formControlName="email" autocomplete="username" />
          </label>
          <label>
            Password
            <input type="password" formControlName="password" autocomplete="current-password" />
          </label>
          @if (error()) {
            <p class="error">{{ error() }}</p>
          }
          <button type="submit" [disabled]="loading()">
            {{ loading() ? 'Logging in…' : 'Log in' }}
          </button>
        </form>
      </div>
    </div>
  `,
  styleUrl: './login.component.css',
})
export class LoginComponent {
  private fb = inject(FormBuilder);
  private auth = inject(AuthService);
  private router = inject(Router);

  form = this.fb.nonNullable.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', Validators.required],
  });

  loading = signal(false);
  error = signal('');

  async submit(): Promise<void> {
    if (this.form.invalid) return;
    this.error.set('');
    this.loading.set(true);
    try {
      const { email, password } = this.form.getRawValue();
      await this.auth.login(email, password);
      this.router.navigate(['/']);
    } catch (err: any) {
      this.error.set(err?.error?.trim() || err?.message || 'Login failed');
    } finally {
      this.loading.set(false);
    }
  }
}
