import { Component, inject } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet, Router } from '@angular/router';
import { AuthService } from './core/auth.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    @if (auth.isLoggedIn()) {
      <nav class="sidebar">
        <div class="sidebar-brand">CodeArena Admin</div>
        <a routerLink="/problems" routerLinkActive="active">Problems</a>
        <a routerLink="/users" routerLinkActive="active">Users</a>
        <a routerLink="/submissions" routerLinkActive="active">Submissions</a>
        <button class="logout-btn" (click)="logout()">Log out</button>
      </nav>
      <main class="main">
        <router-outlet />
      </main>
    } @else {
      <router-outlet />
    }
  `,
  styleUrl: './app.css',
})
export class App {
  auth = inject(AuthService);
  private router = inject(Router);

  logout(): void {
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}
