    import * as v from 'valibot';

export const	RegisterSchema = v.pipe(
  v.object({
            full_name: v.pipe(v.string(),
                v.trim(),
                v.minLength(2, 'Nama lengkap minimal 2 karakter'),
                v.maxLength(255)
            ),
            email: v.pipe(
                v.string(),
                v.trim(),
                v.email('Format email tidak valid'),
                v.maxLength(255)
            ),
            password: v.pipe(
                v.string(),
                v.minLength(8, 'Password minimal 8 karakter'),
                v.maxLength(255)
            ),
            confirm_password: v.string()
        }),
        // Validasi konfirmasi password harus sama persis
        v.forward(
            v.partialCheck(
                [['password'], ['confirm_password']],
                (input) => input.password === input.confirm_password,
                'Konfirmasi password tidak cocok'
            ),
            ['confirm_password']
        )
);

export const LoginSchema = v.object({
      email: v.pipe(
          v.string('Email wajib diisi'),
          v.trim(),
          v.nonEmpty('Email tidak boleh kosong'),
          v.email('Format email tidak valid'),
          v.maxLength(255, 'Email maksimal 255 karakter')
      ),
      password: v.pipe(
          v.string('Password wajib diisi'),
          v.nonEmpty('Password tidak boleh kosong'),
          v.minLength(8, 'Password minimal 8 karakter'),
          v.maxLength(255, 'Password maksimal 255 karakter')
      )
  });

export const ForgotPasswordSchema = v.object({
  email: v.pipe(
    v.string('Email wajib diisi'),
    v.trim(),
    v.nonEmpty('Email tidak boleh kosong'),
    v.email('Format email tidak valid'),
    v.maxLength(255, 'Email maksimal 255 karakter')
  )
});

export const ResendVerificationSchema = v.object({
  email: v.pipe(
    v.string('Email wajib diisi'),
    v.trim(),
    v.nonEmpty('Email tidak boleh kosong'),
    v.email('Format email tidak valid'),
    v.maxLength(255, 'Email maksimal 255 karakter')
  )
});

export const ResetPasswordSchema = v.pipe(
  v.object({
    token: v.pipe(
      v.string('Token reset wajib disertakan'),
      v.trim(),
      v.nonEmpty('Token reset tidak boleh kosong')
    ),
    new_password: v.pipe(
      v.string('Password baru wajib diisi'),
      v.nonEmpty('Password baru tidak boleh kosong'),
      v.minLength(8, 'Password baru minimal 8 karakter'),
      v.maxLength(255, 'Password baru maksimal 255 karakter')
    ),
    confirm_new_password: v.pipe(
      v.string('Konfirmasi password wajib diisi'),
      v.nonEmpty('Konfirmasi password tidak boleh kosong')
    )
  }),
  v.forward(
    v.partialCheck(
      [['new_password'], ['confirm_new_password']],
      (input) => input.new_password === input.confirm_new_password,
      'Konfirmasi password tidak cocok'
    ),
    ['confirm_new_password']
  )
);

export type RegisterInput = v.InferInput<typeof RegisterSchema>;
export type LoginInput = v.InferInput<typeof LoginSchema>;
export type ForgotPasswordInput = v.InferInput<typeof ForgotPasswordSchema>;
export type ResendVerificationInput = v.InferInput<typeof ResendVerificationSchema>;
export type ResetPasswordInput = v.InferInput<typeof ResetPasswordSchema>;
