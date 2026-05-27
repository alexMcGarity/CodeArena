import { Routes } from '@angular/router';
import { adminGuard } from './core/auth.guard';

export const routes: Routes = [
  { path: 'login', loadComponent: () => import('./pages/login/login.component').then(m => m.LoginComponent) },
  {
    path: '',
    canActivate: [adminGuard],
    children: [
      { path: '', redirectTo: 'problems', pathMatch: 'full' },
      { path: 'problems', loadComponent: () => import('./pages/problems/problems.component').then(m => m.ProblemsComponent) },
      { path: 'problems/:id/edit', loadComponent: () => import('./pages/problem-edit/problem-edit.component').then(m => m.ProblemEditComponent) },
      { path: 'problems/new', loadComponent: () => import('./pages/problem-edit/problem-edit.component').then(m => m.ProblemEditComponent) },
      { path: 'users', loadComponent: () => import('./pages/users/users.component').then(m => m.UsersComponent) },
      { path: 'submissions', loadComponent: () => import('./pages/submissions/submissions.component').then(m => m.SubmissionsComponent) },
    ],
  },
  { path: '**', redirectTo: '' },
];
